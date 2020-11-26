import io
import json
import math
import numpy
import os
import os.path
import skimage.io
import struct
import sys

def eprint(s):
	sys.stderr.write(str(s) + "\n")
	sys.stderr.flush()

def per_frame_decorate(f):
	def wrap(*args):
		job_desc = args[0]
		if job_desc['type'] == 'finish':
			output_data_finish(job_desc['key'], job_desc['key'])
			return
		elif job_desc['type'] != 'job':
			return

		args = args[1:]
		outputs = []
		for i in range(len(args[0])):
			inputs = [arg[i] for arg in args]
			output = f(*inputs)
			if not isinstance(output, tuple):
				output = (output,)
			outputs.append(output)
		stack_outputs = []
		for i, t in enumerate(meta['OutputTypes']):
			stacked = [output[i] for output in outputs]
			if t == 'image' or t == 'video':
				stacked = numpy.stack(stacked)
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
			for i, l in enumerate(all_inputs):
				if isinstance(l[0], list):
					all_inputs[i] = [x for arg in l for x in arg]
				else:
					all_inputs[i] = numpy.concatenate(l, axis=0)
			outputs = f(*all_inputs)
			if not isinstance(outputs, tuple):
				outputs = (outputs,)
			output_datas(job_desc['key'], job_desc['key'], len(outputs[0]), outputs)
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
