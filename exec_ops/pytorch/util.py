import json
import numpy
import skimage.io, skimage.transform
import torch

import skyhook_pylib as lib

def read_input(t, path, metadata, format):
	if t == 'image':
		return skimage.io.imread(path)
	else:
		with open(path, 'r') as f:
			data = json.load(f)

		# transform to stream JSON format if needed
		if t == 'shape':
			data = {
				'Shapes': data,
				'Metadata': json.loads(metadata),
			}
		elif t == 'detection':
			data = {
				'Detections': data,
				'Metadata': json.loads(metadata),
			}

		data = lib.data_index(t, data, 0)
		return data

def prepare_input(t, data, opt):
	if t == 'image' or t == 'video':
		im = data
		if 'Width' in opt and 'Height' in opt:
			im = skimage.transform.resize(im, [opt['Height'], opt['Width']], preserve_range=True).astype('uint8')
		return im.transpose(2, 0, 1)
	elif t == 'int':
		return numpy.array(data, dtype='int32')
	elif t == 'floats':
		return numpy.array(data, dtype='float32')
	elif t == 'shape':
		# we will normalize the points by the canvas dims
		dims = data['Metadata']['CanvasDims']

		# encode as 3-tuple: (# shapes, # points in each shape, flat points concat across the shapes)
		counts = []
		points = []
		for i, shape in enumerate(data['Shapes']):
			counts.append(len(shape['Points']))
			for p in shape['Points']:
				p = (float(p[0])/dims[0], float(p[1])/dims[1])
				points.append(p)

		counts = numpy.array(counts, dtype='int32')
		points = numpy.array(points, dtype='float32')
		return (len(data['Shapes']), counts, points)
	elif t == 'detection':
		# we will normalize the points by the canvas dims
		dims = data['Metadata']['CanvasDims']

		# encode as 2-tuple: (# detections, then flat bboxes)
		count = len(data['Detections'])
		boxes = numpy.zeros((count, 4), dtype='float32')
		for i, d in enumerate(data['Detections']):
			boxes[i, :] = [
				float(d['Left'])/dims[0],
				float(d['Top'])/dims[1],
				float(d['Right'])/dims[0],
				float(d['Bottom'])/dims[1],
			]

		return (count, boxes)

	raise Exception('unknown type {}'.format(t))

def collate(t, data_list):
	if t == 'shape':
		shape_counts, point_counts, points = list(zip(*data_list))
		return (
			torch.from_numpy(numpy.array(shape_counts, dtype='int32')),
			torch.cat([torch.from_numpy(x) for x in point_counts], dim=0),
			torch.cat([torch.from_numpy(x) for x in points], dim=0),
		)
	elif t == 'detection':
		counts, boxes = list(zip(*data_list))
		return (
			torch.from_numpy(numpy.array(counts, dtype='int32')),
			torch.cat([torch.from_numpy(x) for x in boxes], dim=0),
		)
	else:
		return torch.stack([torch.from_numpy(data) for data in data_list], dim=0)

def inputs_to_device(inputs, device):
	for i, d in enumerate(inputs):
		if isinstance(d, tuple):
			inputs[i] = [x.to(device) for x in d]
		else:
			inputs[i] = d.to(device)
