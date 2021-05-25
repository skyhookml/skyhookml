import sys
sys.path.append('./python')
import skyhook.common as lib
import skyhook.io
from skyhook.op.op import Operator

import io
import json
import torch.multiprocessing as multiprocessing
import numpy
import os, os.path
import random
import requests
import skimage.io, skimage.transform
import threading
import time
import torch

import skyhook.pytorch.model as model
import skyhook.pytorch.util as util

in_dataset_id = int(sys.argv[1])
params_arg = sys.argv[2]
# TODO: make Python /synchronized-reader endpoint accept batch size
# TODO: make batch size configurable and have auto-reduce-batch-size option
#batch_size = 16

params = json.loads(params_arg)

# For inter-task parallelism, we have:
# - Multiple ingest workers that each read data for one task at a time.
# - A single inference thread that applies the model on prepared inputs.
# - Multiple egress workers that write the outputs to local Go HTTP server.
# Ingest/egress workers operate in pairs, with same worker ID.

def ingress_worker(worker_id, params, operator, task_queue, infer_queue):
	input_options = {}
	for spec in params['InputOptions']:
		input_options[spec['Idx']] = json.loads(spec['Value'])

	while True:
		job = task_queue.get()
		request_id = job['RequestID']
		task = job['Task']

		items = [item_list[0] for item_list in task['Items']['inputs']]
		in_metadatas = []
		for item in items:
			if item['Metadata']:
				in_metadatas.append(json.loads(item['Metadata']))
			else:
				in_metadatas.append({})

		# We optimize inference over video data by handling input options in ffmpeg.
		# Here, we loop over items and update metadata to match the desired resolution.
		# We also get the framerate of the first video input (if any).
		output_defaults = {}
		for i, item in enumerate(items):
			if item['Dataset']['DataType'] != 'video':
				continue
			opt = input_options.get(i, {})
			orig_dims = in_metadatas[i]['Dims']
			new_dims = util.get_resize_dims(orig_dims, opt)
			in_metadatas[i]['Dims'] = new_dims
			item['Metadata'] = json.dumps(in_metadatas[i])
			if 'framerate' not in output_defaults:
				output_defaults['framerate'] = in_metadatas[i]['Framerate']

		rd_resp = requests.post(operator.local_url + '/synchronized-reader', json=items, stream=True)
		rd_resp.raise_for_status()

		in_dtypes = [ds['DataType'] for ds in operator.inputs]

		# Whether we have sent initialization info to the egress worker about this task yet.
		initialized_egress = False

		while True:
			# Read a batch.
			try:
				datas = skyhook.io.read_datas(rd_resp.raw, in_dtypes, in_metadatas)
			except EOFError:
				break

			# Convert datas to our input form.
			# Also get default canvas_dims based on dimensions of the first input image/video/array.
			data_len = lib.data_len(in_dtypes[0], datas[0])
			pytorch_datas = []
			for ds_idx, data in enumerate(datas):
				t = in_dtypes[ds_idx]

				if t == 'video':
					# We already handled input options by mutating the item metadata.
					# So, here, we just need to transpose.
					pytorch_data = torch.from_numpy(data).permute(0, 3, 1, 2)
				else:
					opt = input_options.get(ds_idx, {})
					cur_pytorch_datas = []
					for i in range(data_len):
						element = lib.data_index(t, data, i)
						pytorch_data = util.prepare_input(t, element, in_metadatas[ds_idx], opt)
						cur_pytorch_datas.append(pytorch_data)
					pytorch_data = util.collate(t, cur_pytorch_datas)

				if 'canvas_dims' not in output_defaults and (t == 'image' or t == 'video' or t == 'array'):
					output_defaults['canvas_dims'] = [pytorch_data.shape[3], pytorch_data.shape[2]]

				pytorch_datas.append(pytorch_data)

			# Initialize the egress worker if not done already.
			# We send this through the inference thread to synchronize with any previous closed inference jobs and such.
			if not initialized_egress:
				infer_queue.put({
					'Type': 'init',
					'WorkerID': worker_id,
					'RequestID': request_id,
					'Task': task,
					'OutputDefaults': output_defaults,
				})
				initialized_egress = True

			# Pass on to inference thread.
			infer_queue.put({
				'Type': 'infer',
				'WorkerID': worker_id,
				'Datas': pytorch_datas,
			})

		# Close the egress worker.
		# We send this through the inference thread so that any pending inference jobs for this task finish first.
		infer_queue.put({
			'Type': 'close',
			'WorkerID': worker_id,
		})

def infer_thread(in_dataset_id, params, infer_queue, egress_queues):
	device = torch.device('cuda:0')
	cpu_device = torch.device('cpu')
	model_path = 'data/items/{}/model.pt'.format(in_dataset_id)
	save_dict = torch.load(model_path)

	# overwrite parameters in save_dict['arch'] with parameters from
	# params['Components'][comp_idx]
	arch = save_dict['arch']
	if params.get('Components', None):
		overwrite_comp_params = {int(k): v for k, v in params['Components'].items()}
		for comp_idx, comp_spec in enumerate(arch['Components']):
			comp_params = {}
			if comp_spec['Params']:
				comp_params = json.loads(comp_spec['Params'])
			if overwrite_comp_params.get(comp_idx, None):
				comp_params.update(json.loads(overwrite_comp_params[comp_idx]))
			comp_spec['Params'] = json.dumps(comp_params)

	example_inputs = save_dict['example_inputs']
	util.inputs_to_device(example_inputs, device)

	net = model.Net(arch, save_dict['comps'], example_inputs, save_dict['example_metadatas'], output_datasets=params['OutputDatasets'], infer=True, device=device)
	net.to(device)

	net.load_state_dict(save_dict['model'])
	net.eval()

	with torch.no_grad():
		while True:
			job = infer_queue.get()
			worker_id = job['WorkerID']
			if job['Type'] != 'infer':
				# For close job, simply forward it to the egress worker.
				egress_queues[worker_id].put(job)
				continue
			pytorch_datas = job['Datas']

			# Apply the model.
			util.inputs_to_device(pytorch_datas, device)
			y = net(*pytorch_datas)
			y = list(y)
			util.inputs_to_device(y, cpu_device)
			egress_queues[worker_id].put({
				'Type': 'infer',
				'Data': y,
			})

def egress_worker(worker_id, operator, egress_queue):
	out_dtypes = [ds['DataType'] for ds in operator.outputs]

	while True:
		job = egress_queue.get()
		if job['Type'] != 'init':
			raise Exception('egress: expected init job but got {}'.format(job['Type']))

		request_id = job['RequestID']
		task = job['Task']
		output_defaults = job['OutputDefaults']
		canvas_dims = output_defaults.get('canvas_dims', [1280, 720])
		framerate = output_defaults.get('framerate', [10, 1])

		def gen():
			sent_meta = False

			while True:
				job = egress_queue.get()
				if job['Type'] == 'close':
					return
				elif job['Type'] != 'infer':
					raise Exception('egress: expected init or infer job but got {}'.format(job['Type']))

				y = job['Data']

				# Convert back from pytorch to skyhookml stream data format.
				out_datas = []
				out_metadatas = []
				for out_idx, t in enumerate(out_dtypes):
					cur = y[out_idx]
					if t in ['image', 'video', 'array']:
						cur = cur.permute(0, 2, 3, 1).numpy()
						out_datas.append(cur)
						if t == 'image':
							out_metadatas.append({})
						elif t == 'video':
							out_metadatas.append({
								'Framerate': framerate,
								'Dims': [cur.shape[2], cur.shape[1]],
							})
						elif t == 'array':
							out_metadatas.append({
								'Width': cur.shape[2],
								'Height': cur.shape[1],
								'Channels': cur.shape[3],
								'Type': cur.dtype.name,
							})
					elif t == 'detection':
						# detections are represented as a dict
						# - cur['counts'] is # detections in each image
						# - cur['detections'] is the flat list of detections (cls, xyxy, conf)
						# - cur['categories'] is optional string category list
						# first, convert from boxes to skyhookml detections
						flat_detections = []
						for box in cur['detections'].tolist():
							cls, left, top, right, bottom, conf = box
							if 'categories' in cur:
								category = cur['categories'][int(cls)]
							else:
								category = 'category{}'.format(int(cls))
							flat_detections.append({
								'Left': int(left*canvas_dims[0]),
								'Top': int(top*canvas_dims[1]),
								'Right': int(right*canvas_dims[0]),
								'Bottom': int(bottom*canvas_dims[1]),
								'Score': float(conf),
								'Category': category,
							})
						# second, group up the boxes
						prefix_sum = 0
						detections = []
						for count in cur['counts']:
							detections.append(flat_detections[prefix_sum:prefix_sum+count])
							prefix_sum += count
						out_datas.append(detections)
						out_metadatas.append({
							'CanvasDims': canvas_dims,
						})
					else:
						out_datas.append(cur.tolist())
						out_metadatas.append({})

				# If we haven't sent the meta packet yet, send it.
				# We delay until here so that we have metadatas.
				if not sent_meta:
					sent_meta = True
					metas = []
					for out_idx, ds in enumerate(operator.outputs):
						metas.append({
							'Dataset': ds,
							'Key': task['Key'],
							'Metadata': json.dumps(out_metadatas[out_idx]),
						})
					buf = io.BytesIO()
					skyhook.io.write_json(buf, metas)
					yield buf.getvalue()

				# Encode outputs and yield the bytes.
				buf = io.BytesIO()
				skyhook.io.write_datas(buf, out_dtypes, out_datas)
				yield buf.getvalue()

		# Make the build request.
		requests.post(operator.local_url + '/build', data=gen()).raise_for_status()

		# Since this is async, here we have to write the skjson line responding to the task.
		print('skjson'+json.dumps({'RequestID': request_id}), flush=True)

# Watch for child processes that failed in separate thread.
def watchdog(operator):
	while True:
		time.sleep(1)
		failed = False
		for p in operator.plist:
			if not p.is_alive():
				failed = True
		if not failed:
			continue

		print('watchdog: a child process died, terminating', flush=True)
		operator.close()
		os._exit(-1)
		break

class InferOperator(Operator):
	def __init__(self, meta_packet):
		super(InferOperator, self).__init__(meta_packet)

		# Use more than one parallelism since ffmpeg on input side may be bottleneck.
		self.nthreads = max(1, min(4, os.cpu_count()//2))

		# Queue from operator.apply to ingress worker about tasks.
		self.task_queue = multiprocessing.Queue(1)
		# Queue from ingress workers to inference thread.
		infer_queue = multiprocessing.Queue(1)
		# Queues from inference thread to egress workers.
		egress_queues = []

		self.plist = []

		for worker_id in range(self.nthreads):
			egress_queue = multiprocessing.Queue(1)
			egress_queues.append(egress_queue)
			p = multiprocessing.Process(target=ingress_worker, args=(worker_id, params, self, self.task_queue, infer_queue))
			p.start()
			self.plist.append(p)
			p = multiprocessing.Process(target=egress_worker, args=(worker_id, self, egress_queue))
			p.start()
			self.plist.append(p)

		p = multiprocessing.Process(target=infer_thread, args=(in_dataset_id, params, infer_queue, egress_queues))
		p.start()
		self.plist.append(p)
		threading.Thread(target=watchdog, args=(self,)).start()

	def parallelism(self):
		return self.nthreads

	def apply(self, request_id, task):
		self.task_queue.put({
			'RequestID': request_id,
			'Task': task,
		})

	def close(self):
		for p in self.plist:
			p.kill()

lib.run(InferOperator, async_apply=True)
