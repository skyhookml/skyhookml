import sys
sys.path.append('./python')
import skyhook.common as lib

import json
import numpy
import os, os.path
import random
import requests
import skimage.io, skimage.transform

import torch

import skyhook.pytorch.model as model
import skyhook.pytorch.util as util

in_dataset_id = int(sys.argv[1])
params_arg = sys.argv[2]
batch_size = 16

params = json.loads(params_arg)

device = torch.device('cuda:0')
#device = torch.device('cpu')
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

input_options = {}
for spec in params['InputOptions']:
	input_options[spec['Idx']] = json.loads(spec['Value'])

meta = None
def meta_func(x):
	global meta
	meta = x

@torch.no_grad()
def callback_func(*args):
	job_desc = args[0]
	args = args[1:]
	if job_desc['type'] == 'finish':
		lib.output_data_finish(job_desc['key'], job_desc['key'])
		return
	elif job_desc['type'] != 'job':
		return

	input_len = lib.data_len(meta['InputTypes'][0], args[0])
	# process the inputs one batch size at a time
	for inp_start in range(0, input_len, batch_size):
		inp_end = min(inp_start+batch_size, input_len)

		# find the dimensions of the first input image
		# we currently use this to fill canvas_dims of detection/shape outputs
		canvas_dims = None

		# get the slice corresponding to current batch from args
		# and convert it to our pytorch input form
		datas = []
		for ds_idx, arg in enumerate(args):
			t = meta['InputTypes'][ds_idx]

			if t == 'video':
				# we optimize inference over video by handling input options in golang
				# so here we just need to transpose
				data = torch.from_numpy(arg[inp_start:inp_end, :, :, :]).permute(0, 3, 1, 2)
			else:
				opts = input_options.get(ds_idx, {})
				cur_datas = []
				for i in range(inp_start, inp_end):
					input = lib.data_index(t, arg, i)
					data = util.prepare_input(t, input, opts)
					if canvas_dims is None and (t == 'image' or t == 'video' or t == 'array'):
						canvas_dims = [data.shape[2], data.shape[1]]
					cur_datas.append(data)
				data = util.collate(t, cur_datas)

			datas.append(data)
		if not canvas_dims:
			canvas_dims = [1280, 720]

		# process this batch through the model
		util.inputs_to_device(datas, device)
		y = net(*datas)

		# extract and emit outputs
		out_datas = []
		for out_idx, t in enumerate(meta['OutputTypes']):
			cur = y[out_idx]
			if t in ['image', 'video', 'array']:
				out_datas.append(cur.permute(0, 2, 3, 1).cpu().numpy())
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
				out_datas.append({
					'Detections': detections,
					'Metadata': {
						'CanvasDims': canvas_dims,
					},
				})
			elif t == 'int':
				out_datas.append({
					'Ints': cur.tolist(),
				})
			else:
				out_datas.append(cur.tolist())
		lib.output_datas(job_desc['key'], job_desc['key'], out_datas)

lib.run(callback_func, meta_func)
