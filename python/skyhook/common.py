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
import skyhook.io

def eprint(s):
	sys.stderr.write(str(s) + "\n")
	sys.stderr.flush()

# sometimes JSON that we input ends up containing null (=> None) entries instead of list
# this helper restores lists where lists are expected
def non_null_list(l):
	if l is None:
		return []
	return l

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
	if t == 'int':
		return {
			'Ints': data['Ints'][i],
			'Metadata': data['Metadata'],
		}
	else:
		return data[i]

# stack a bunch of individual data (like data_index output)
def data_stack(t, datas):
	if t == 'image' or t == 'video' or t == 'array':
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
	elif t == 'int':
		return {
			'Ints': [data['Ints'] for data in datas],
			'Metadata': datas[0].get('Metadata', {}),
		}
	else:
		return datas

# stack a bunch of regular data
# this fails for non-sequence data, unless len(datas)==1, in which case it simply returns the data
def data_concat(t, datas):
	if len(datas) == 1:
		return datas[0]

	if t == 'image' or t == 'video' or t == 'array':
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
	elif t == 'int':
		return {
			'Ints': [x for data in datas for x in data['Ints']],
			'Metadata': datas[0].get('Metadata', {}),
		}
	else:
		return [x for data in datas for x in data]

def data_len(t, data):
	if t == 'shape':
		return len(data['Shapes'])
	if t == 'detection':
		return len(data['Detections'])
	if t == 'int':
		return len(data['Ints'])
	return len(data)

# Load data from disk.
# The output corresponds to what we would get from input_datas.
# It can be passed to data_index, data_concat, data_len, etc.
def load_item(dataset, item):
	fname = 'data/items/{}/{}.{}'.format(dataset['ID'], item['Key'], item['Ext'])
	t = dataset['DataType']
	metadata, format = item['Metadata'], item['Format']

	if t == 'image':
		im = skimage.io.imread(fname)
		return [im]
	elif t == 'video':
		raise Exception('load_item cannot handle video data')
	elif t == 'array':
		metadata = json.loads(metadata)
		dt = numpy.dtype(metadata['Type'])
		dt = dt.newbyteorder('>')
		return numpy.fromfile(fname, dtype=dt).reshape(-1, metadata['Height'], metadata['Width'], metadata['Channels'])
	else:
		with open(fname, 'r') as f:
			data = json.load(f)

		# transform to stream JSON format if needed
		if t == 'shape':
			data = [non_null_list(l) for l in data]
			data = {
				'Shapes': data,
				'Metadata': json.loads(metadata),
			}
		elif t == 'detection':
			data = [non_null_list(l) for l in data]
			data = {
				'Detections': data,
				'Metadata': json.loads(metadata),
			}
		elif t == 'int':
			data = {
				'Ints': data,
				'Metadata': json.loads(metadata),
			}

		return data

def load_video(dataset, item):
	fname = 'data/items/{}/{}.{}'.format(dataset['ID'], item['Key'], item['Ext'])
	metadata = json.loads(item['Metadata'])
	return ffmpeg.Ffmpeg(fname, metadata['Dims'], metadata['Framerate'])

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
		output_datas(job_desc['key'], job_desc['key'], stack_outputs)
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
			output_datas(job_desc['key'], job_desc['key'], outputs)
			output_data_finish(job_desc['key'], job_desc['key'])
	return wrap

stdin = None
stdout = None
meta = None

def input_json():
	try:
		return skyhook.io.read_json(stdin)
	except EOFError:
		return None

def input_datas():
	return skyhook.io.read_datas(stdin, meta['InputTypes'])

def output_json(x):
	skyhook.io.write_json(stdout, x)

def output_datas(in_key, key, datas):
	output_json({
		'Type': 'data_data',
		'Key': in_key,
		'OutputKey': key,
	})
	skyhook.io.write_datas(stdout, meta['OutputTypes'], datas)
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
