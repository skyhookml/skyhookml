import sys
sys.path.append('./')
import skyhook_pylib as lib

import json
import numpy
import os, os.path
import random
import requests
import skimage.io, skimage.transform

import torch

import model
import util

node_id = int(sys.argv[1])
params_arg = sys.argv[2]

params = json.loads(params_arg)

device = torch.device('cuda:0')
#device = torch.device('cpu')
model_path = 'models/{}.pt'.format(node_id)
save_dict = torch.load(model_path)
net = model.Net(save_dict['arch'], save_dict['comps'], save_dict['example_inputs'], output_datasets=params['OutputDatasets'])
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
	for i, input in enumerate(inputs):
		t = meta['InputTypes'][i]
		opts = input_options.get(i, {})
		data = util.prepare_input(t, input, opts)
		data = util.collate(t, [data])
		datas.append(data)
	util.inputs_to_device(datas, device)
	y = net(*datas)

	y_ = []
	for i, t in enumerate(meta['OutputTypes']):
		cur = y[i][0]
		if t in ['image', 'video']:
			y_.append(cur.cpu().numpy().transpose(1, 2, 0))
		elif t == 'detection':
			detections = []
			for box in cur.tolist():
				left, top, right, bottom, conf, cls = box
				if conf < 0.2:
					continue
				detections.append({
					'Left': int(left*1280//320),
					'Top': int(top*720//192),
					'Right': int(right*1280//320),
					'Bottom': int(bottom*720/192),
				})
			y_.append({
				'Detections': detections,
				'Metadata': {
					'CanvasDims': [1280, 720],
				},
			})
		else:
			y_.append(cur.tolist())
	return tuple(y_)

lib.run(callback_func, meta_func)
