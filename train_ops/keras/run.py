import json
import keras.backend, keras.layers, keras.models
import numpy
import os, os.path
import random
import requests
import skimage.io, skimage.transform
import sys
import tensorflow as tf

import model

import sys
sys.path.append('./')
import skyhook_pylib as lib

node_id = int(sys.argv[1])
params_arg = sys.argv[2]
archs_arg = sys.argv[3]
exec_params_arg = sys.argv[4]

params = json.loads(params_arg)
archs = json.loads(archs_arg)
exec_params = json.loads(exec_params_arg)

m, _ = model.get_model(archs, params)

model_path = 'models/{}.h5'.format(node_id)
m.load_weights(model_path, by_name=True)

meta = None
def meta_func(x):
	global meta
	meta = x

@lib.per_frame_decorate
def callback_func(*inputs):
	inputs = [[input] for input in inputs]
	inputs[0][0] = skimage.transform.resize(inputs[0][0], [256, 256], preserve_range=True).astype('uint8')
	y = m.predict(inputs)

	if not isinstance(y, list):
		y = [y]

	y_ = []
	for i, t in enumerate(meta['OutputTypes']):
		if t in ['image', 'video']:
			y_.append(y[i])
		else:
			y_.append(y[i].argmax().tolist())
	return tuple(y_)

lib.run(callback_func, meta_func)
