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
	return data[i]

# stack a bunch of individual data (like data_index output)
def data_stack(t, datas):
	if t == 'image' or t == 'video' or t == 'array':
		return numpy.stack(datas)
	else:
		return datas

# stack a bunch of regular data
# this fails for non-sequence data, unless len(datas)==1, in which case it simply returns the data
def data_concat(t, datas):
	if len(datas) == 1:
		return datas[0]

	if t == 'image' or t == 'video' or t == 'array':
		return numpy.concatenate(datas, axis=0)
	else:
		return [x for data in datas for x in data]

def data_len(t, data):
	return len(data)

def decode_metadata(dataset, item):
	metadata = {}
	if dataset['Metadata']:
		metadata.update(json.loads(dataset['Metadata']))
	if item['Metadata']:
		metadata.update(json.loads(item['Metadata']))
	return metadata

# Load data from disk.
# The output corresponds to what we would get from input_datas.
# It can be passed to data_index, data_concat, data_len, etc.
def load_item(dataset, item):
	fname = 'data/items/{}/{}.{}'.format(dataset['ID'], item['Key'], item['Ext'])
	t = dataset['DataType']
	metadata = decode_metadata(dataset, item)
	format = item['Format']

	if t == 'image':
		im = skimage.io.imread(fname)
		return [im]
	elif t == 'video':
		raise Exception('load_item cannot handle video data')
	elif t == 'array':
		dt = numpy.dtype(metadata['Type'])
		dt = dt.newbyteorder('>')
		return numpy.fromfile(fname, dtype=dt).reshape(-1, metadata['Height'], metadata['Width'], metadata['Channels'])
	else:
		with open(fname, 'r') as f:
			data = json.load(f)

		# Correct cases where Golang encodes nil slice as "null" instead of list.
		for i in range(len(data)):
			if data[i] is None:
				data[i] = []

		return data

def load_video(dataset, item):
	fname = 'data/items/{}/{}.{}'.format(dataset['ID'], item['Key'], item['Ext'])
	metadata = decode_metadata(dataset, item)
	return ffmpeg.Ffmpeg(fname, metadata['Dims'], metadata['Framerate'])

def get_tracks(detections):
	track_dict = {}
	for frame_idx, dlist in enumerate(detections):
		for detection in dlist:
			detection['FrameIdx'] = frame_idx
			track_id = detection['TrackID']
			if track_id not in track_dict:
				track_dict[track_id] = []
			track_dict[track_id].append(detection)
	return track_dict.values()

def tracks_to_detections(tracks, nframes):
	detections = [[] for _ in range(nframes)]
	for track in tracks:
		for detection in track:
			detections[detection['FrameIdx']].append(detection)
	return detections

def run(operator_provider, async_apply=False):
	import importlib
	import json

	stdin = sys.stdin.detach()
	meta = skyhook.io.read_json(stdin)
	operator = operator_provider(meta)

	while True:
		try:
			request = skyhook.io.read_json(stdin)
		except EOFError:
			break

		id = request['RequestID']
		name = request['Name']
		if request['JSON']:
			params = json.loads(request['JSON'])
		else:
			params = None

		response = None
		if name == 'parallelism':
			response = operator.parallelism()
		elif name == 'get_tasks':
			response = operator.get_tasks(params)
		elif name == 'apply':
			if async_apply:
				operator.apply(id, params)
				continue

			operator.apply(params)

		packet = {
			'RequestID': id,
		}
		if response is not None:
			packet['JSON'] = json.dumps(response)
		print('skjson'+json.dumps(packet), flush=True)

	operator.close()
