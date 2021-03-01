import io
import json
import math
import numpy
import os
import os.path
import skimage.io
import struct
import sys

import skyhook.ffmpeg as ffmpeg

def eprint(s):
	sys.stderr.write(str(s) + "\n")
	sys.stderr.flush()

def data_index(t, data, i):
	if t == 'shape':
		return {
			'Shapes': data['Shapes'][i],
			'Metadata': data['Metadata'],
		}
	if t == 'detection':
		return {
			'Detections': data['Detections'][i],
			'Metadata': data['Metadata'],
		}
	else:
		return data[i]

# stack a bunch of individual data (like data_index output)
def data_stack(t, datas):
	if t == 'image' or t == 'video':
		return numpy.stack(datas)
	elif t == 'shape':
		return {
			'Shapes': [data['Shapes'] for data in datas],
			'Metadata': datas[0].get('Metadata', {}),
		}
	elif t == 'detection':
		return {
			'Detections': [data['Detections'] for data in datas],
			'Metadata': datas[0].get('Metadata', {}),
		}
	else:
		return datas

# stack a bunch of regular data
def data_concat(t, datas):
	if t == 'image' or t == 'video':
		return numpy.concatenate(datas, axis=0)
	elif t == 'shape':
		return {
			'Shapes': [shape_list for data in datas for shape_list in data['Shapes']],
			'Metadata': datas[0].get('Metadata', {}),
		}
	elif t == 'detection':
		return {
			'Detections': [detection_list for data in datas for detection_list in data['Detections']],
			'Metadata': datas[0].get('Metadata', {}),
		}
	else:
		return [x for data in datas for x in data]

def data_len(t, data):
	if t == 'shape':
		return len(data['Shapes'])
	if t == 'detection':
		return len(data['Detections'])
	else:
		return len(data)

def load_item(dataset, item):
	fname = 'items/{}/{}.{}'.format(dataset['ID'], item['Key'], item['Ext'])
	t = dataset['DataType']
	if t == 'image':
		return skimage.io.imread(fname)
	elif t == 'video':
		metadata = json.loads(item['Metadata'])
		return ffmpeg.Ffmpeg(fname, metadata['Dims'], metadata['Framerate'])
	else:
		with open(fname, 'r') as f:
			data = json.load(f)
		return data

def per_frame_decorate(f):
	def wrap(*args):
		job_desc = args[0]
		if job_desc['type'] == 'finish':
			output_data_finish(job_desc['key'], job_desc['key'])
			return
		elif job_desc['type'] != 'job':
			return

		args = args[1:]
		input_len = data_len(meta['InputTypes'][0], args[0])
		outputs = []
		for i in range(input_len):
			inputs = [data_index(meta['InputTypes'][ds_idx], arg, i) for ds_idx, arg in enumerate(args)]
			output = f(*inputs)
			if not isinstance(output, tuple):
				output = (output,)
			outputs.append(output)
		stack_outputs = []
		for i, t in enumerate(meta['OutputTypes']):
			stacked = data_stack(t, [output[i] for output in outputs])
			stack_outputs.append(stacked)
		output_datas(job_desc['key'], job_desc['key'], len(outputs), stack_outputs)
	return wrap

def all_decorate(f):
	def wrap(*args):
		job_desc = args[0]
		all_inputs = job_desc['state']
		if job_desc['type'] == 'job':
			args = args[1:]
			if all_inputs is None:
				all_inputs = [[arg] for arg in args]
			else:
				for i, arg in enumerate(args):
					all_inputs[i].append(arg)
			return all_inputs
		elif job_desc['type'] == 'finish':
			all_inputs = [data_concat(meta['InputTypes'][ds_idx], datas) for ds_idx, datas in enumerate(all_inputs)]
			outputs = f(*all_inputs)
			if not isinstance(outputs, tuple):
				outputs = (outputs,)
			output_len = data_len(meta['OutputTypes'][0], outputs[0])
			output_datas(job_desc['key'], job_desc['key'], output_len, outputs)
			output_data_finish(job_desc['key'], job_desc['key'])
	return wrap

stdin = None
stdout = None
meta = None

def input_json():
	buf = stdin.read(4)
	if not buf:
		return None
	(hlen,) = struct.unpack('>I', buf[0:4])
	json_data = stdin.read(hlen)
	return json.loads(json_data.decode('utf-8'))

def input_video():
	header = input_json()
	size = header['Length']*header['Width']*header['Height']*3
	buf = stdin.read(size)
	return numpy.frombuffer(buf, dtype='uint8').reshape((header['Length'], header['Height'], header['Width'], 3))

def input_datas():
	datas = []
	for t in meta['InputTypes']:
		if t == 'image' or t == 'video':
			datas.append(input_video())
		else:
			datas.append(input_json())
	return datas

def output_json(x):
	s = json.dumps(x).encode()
	stdout.write(struct.pack('>I', len(s)))
	stdout.write(s)

def output_video(x):
	output_json({
		'Length': x.shape[0],
		'Width': x.shape[2],
		'Height': x.shape[1],
	})
	stdout.write(x.tobytes())

def output_datas(in_key, key, l, datas):
	output_json({
		'Type': 'data_data',
		'Key': in_key,
		'OutputKey': key,
		'Length': l,
	})
	for i, t in enumerate(meta['OutputTypes']):
		if t == 'image' or t == 'video':
			output_video(datas[i])
		else:
			output_json(datas[i])
	stdout.flush()

def output_data_finish(in_key, key):
	output_json({
		'Type': 'data_finish',
		'Key': in_key,
		'OutputKey': key,
	})
	stdout.flush()

def run(callback_func, meta_func=None):
	global stdin, stdout, meta

	if sys.version_info[0] >= 3:
		stdin = sys.stdin.detach()
		stdout = sys.stdout.buffer
	else:
		stdin = sys.stdin
		stdout = sys.stdout

	meta = input_json()

	if meta_func:
		meta_func(meta)

	states = {}
	while True:
		packet = input_json()
		if packet is None:
			break

		if packet['Type'] == 'init':
			states[packet['Key']] = None
		elif packet['Type'] == 'job':
			# job packet
			key = packet['Key']
			datas = input_datas()
			inputs = [{
				'type': 'job',
				'length': packet['Length'],
				'key': key,
				'state': states[key],
			}] + datas
			states[key] = callback_func(*inputs)
		elif packet['Type'] == 'finish':
			key = packet['Key']
			inputs = [{
				'type': 'finish',
				'key': key,
				'state': states[key],
			}]
			inputs.extend([None]*len(meta['InputTypes']))
			callback_func(*inputs)
			del states[key]
			output_json({
				'Type': 'finish',
				'Key': key,
			})
			stdout.flush()
