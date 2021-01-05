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
arch_arg = sys.argv[3]
comps_arg = sys.argv[4]
exec_params_arg = sys.argv[5]

params = json.loads(params_arg)
arch = json.loads(arch_arg)
comps = json.loads(comps_arg)
exec_params = json.loads(exec_params_arg)

arch = arch['Params']
comps = {int(comp_id): comp['Params'] for comp_id, comp in comps.items()}

device = torch.device('cuda:0')
#device = torch.device('cpu')
model_path = 'models/{}.pt'.format(node_id)
save_dict = torch.load(model_path)
net = model.Net(arch, comps, params, save_dict['example_inputs'])
net.to(device)

net.load_state_dict(save_dict['model'])
net.eval()

meta = None
def meta_func(x):
	global meta
	meta = x

def inputs_to_device(inputs):
	for i, d in enumerate(inputs):
		if isinstance(d, tuple):
			inputs[i] = [x.to(device) for x in d]
		else:
			inputs[i] = d.to(device)

@lib.per_frame_decorate
def callback_func(*inputs):
	datas = []
	for i, input in enumerate(inputs):
		t = meta['InputTypes'][i]
		opts = params['InputDatasets'][i]['Options']
		if opts:
			opts = json.loads(opts)
		else:
			opts = {}
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
