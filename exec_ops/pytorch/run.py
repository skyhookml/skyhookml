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

params = json.loads(params_arg)

device = torch.device('cuda:0')
#device = torch.device('cpu')
model_path = 'items/{}/model.pt'.format(in_dataset_id)
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

net = model.Net(arch, save_dict['comps'], save_dict['example_inputs'], save_dict['example_metadatas'], output_datasets=params['OutputDatasets'], infer=True)
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

@lib.per_frame_decorate
def callback_func(*inputs):
	datas = []
	# find the dimensions of the first input image
	# we currently use this to fill canvas_dims of detection/shape outputs
	canvas_dims = None
	for i, input in enumerate(inputs):
		t = meta['InputTypes'][i]
		opts = input_options.get(i, {})
		data = util.prepare_input(t, input, opts)
		if canvas_dims is None and (t == 'image' or t == 'video' or t == 'array'):
			canvas_dims = [data.shape[2], data.shape[1]]
		data = util.collate(t, [data])
		datas.append(data)
	if not canvas_dims:
		canvas_dims = [1280, 720]
	util.inputs_to_device(datas, device)
	y = net(*datas)

	y_ = []
	for i, t in enumerate(meta['OutputTypes']):
		cur = y[i]
		if t in ['image', 'video', 'array']:
			y_.append(cur[0].cpu().numpy().transpose(1, 2, 0))
		elif t == 'detection':
			# detections are represented as a dict
			# - cur['counts'] is # detections in each image
			# - cur['detections'] is the flat list of detections (cls, xyxy, conf)
			# - cur['categories'] is optional string category list
			detections = []
			for box in cur['detections'].tolist():
				cls, left, top, right, bottom, conf = box
				if 'categories' in cur:
					category = cur['categories'][int(cls)]
				else:
					category = 'category{}'.format(int(cls))
				detections.append({
					'Left': int(left*canvas_dims[0]),
					'Top': int(top*canvas_dims[1]),
					'Right': int(right*canvas_dims[0]),
					'Bottom': int(bottom*canvas_dims[1]),
					'Score': float(conf),
					'Category': category,
				})
			y_.append({
				'Detections': detections,
				'Metadata': {
					'CanvasDims': canvas_dims,
				},
			})
		elif t == 'int':
			y_.append({
				'Ints': cur[0].tolist(),
			})
		else:
			y_.append(cur[0].tolist())
	return tuple(y_)

lib.run(callback_func, meta_func)
