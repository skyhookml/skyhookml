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

node_id = int(sys.argv[1])
url = sys.argv[2]
params_arg = sys.argv[3]
archs_arg = sys.argv[4]
datasets_arg = sys.argv[5]

params = json.loads(params_arg)
archs = json.loads(archs_arg)
datasets = json.loads(datasets_arg)
datasets = {int(ds_id): dataset for ds_id, dataset in datasets.items()}

class MyGenerator(keras.utils.Sequence):
	def __init__(self, inputs, outputs, keys, items, batch_size=1):
		self.inputs = inputs
		self.outputs = outputs
		self.keys = keys
		self.items = items
		self.batch_size = batch_size

	def __len__(self):
		return len(self.keys)//self.batch_size

	def __getitem__(self, idx):
		batch_keys = self.keys[idx*self.batch_size:(idx+1)*self.batch_size]
		items = []
		for i, dataset in enumerate(self.inputs+self.outputs):
			if dataset['DataType'] == 'image':
				ims = numpy.zeros((self.batch_size, 256, 256, 3), dtype='uint8')
				for j, key in enumerate(batch_keys):
					item = self.items[key][i]
					path = 'items/{}/{}.{}'.format(item['Dataset']['ID'], item['ID'], item['Ext'])
					im = skimage.io.imread(path)
					im = skimage.transform.resize(im, [256, 256], preserve_range=True).astype('uint8')
					ims[j, :, :, :] = im
				items.append(ims)
			elif dataset['DataType'] == 'int':
				onehots = numpy.zeros((self.batch_size, 2), dtype='float32')
				for j, key in enumerate(batch_keys):
					item = self.items[key][i]
					path = 'items/{}/{}.{}'.format(item['Dataset']['ID'], item['ID'], item['Ext'])
					with open(path, 'r') as f:
						cls = json.load(f)[0]
					onehots[j, cls] = 1
				items.append(onehots)

		return items[0:len(self.inputs)], items[len(self.inputs):]

def get_generators(inputs, outputs):
	datasets = inputs+outputs

	# get items
	items = {}
	for dataset in datasets:
		ds_id = dataset['ID']
		cur_items = requests.get(url+'/datasets/{}/items'.format(ds_id)).json()
		for item in cur_items:
			key = item['Key']
			if key in items:
				items[key].append(item)
			else:
				items[key] = [item]
	# only keep keys that exist in all datasets
	for key in list(items.keys()):
		if len(items[key]) != len(datasets):
			del items[key]

	keys = list(items.keys())
	random.shuffle(keys)
	num_val = len(keys)//5
	val_keys = keys[0:num_val]
	train_keys = keys[num_val:]

	train_generator = MyGenerator(inputs, outputs, train_keys, items)
	val_generator = MyGenerator(inputs, outputs, val_keys, items)
	return train_generator, val_generator

model, layers = model.get_model(archs, params)

# set trainable
if params.get('TrainLayers', None):
	for name, layer in layers.items():
		layer.trainable = name in params['TrainLayers']

# compile model -- hardcoded for now
losses = {'layer8': 'categorical_crossentropy'}
loss_weights = {'layer8': 1.0}

model.compile(optimizer='adam', loss=losses, loss_weights=loss_weights)

# load operations
if params.get('LoadFrom', None):
	for node_id in params['LoadFrom']:
		model.load_weights('models/{}.h5'.format(node_id), by_name=True)

input_datasets = [datasets[name] for name in params['InputDatasets']]
output_datasets = [datasets[name] for name in params['OutputDatasets']]
train_generator, val_generator = get_generators(input_datasets, output_datasets)
model_path = 'models/{}.h5'.format(node_id)
cb_checkpoint = keras.callbacks.ModelCheckpoint(
	filepath=model_path,
	save_weights_only=True,
	monitor='val_loss',
	mode='min',
	save_best_only=True
)
cb_stop = keras.callbacks.EarlyStopping(
	monitor="val_loss",
	patience=25,
)
model.fit(
	train_generator,
	epochs=1000,
	validation_data=val_generator,
	callbacks=[cb_checkpoint, cb_stop]
)
