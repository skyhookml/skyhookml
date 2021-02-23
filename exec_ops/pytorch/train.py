import sys
sys.path.append('./')
# needed by util.py
import skyhook_pylib as lib

import json
import numpy
import os, os.path
import random
import requests
import skimage.io, skimage.transform

import torch
import torch.optim
import torch.utils

import model
import util

node_id = int(sys.argv[1])
url = sys.argv[2]
params_arg = sys.argv[3]
arch_arg = sys.argv[4]
comps_arg = sys.argv[5]
datasets_arg = sys.argv[6]

params = json.loads(params_arg)
arch = json.loads(arch_arg)
comps = json.loads(comps_arg)
datasets = json.loads(datasets_arg)

arch = arch['Params']
comps = {int(comp_id): comp['Params'] for comp_id, comp in comps.items()}

class MyDataset(torch.utils.data.Dataset):
	def __init__(self, datasets, keys, items):
		self.datasets = datasets
		self.keys = keys
		self.items = items

	def __len__(self):
		return len(self.keys)

	def __getitem__(self, idx):
		if torch.is_tensor(idx):
			idx = idx.tolist()

		key = self.keys[idx]
		items = self.items[key]

		inputs = []
		for dataset in self.datasets:
			item = items[dataset['ID']]
			path = 'items/{}/{}.{}'.format(dataset['ID'], item['Key'], item['Ext'])
			data = util.read_input(dataset['DataType'], path, item['Metadata'], item['Format'])
			data = util.prepare_input(dataset['DataType'], data, dataset['Options'])
			inputs.append(data)

		return inputs

	def collate_fn(self, batch):
		inputs = list(zip(*batch))
		for i, dataset in enumerate(self.datasets):
			inputs[i] = util.collate(dataset['DataType'], inputs[i])
		return inputs

def get_datasets():
	# add options to datasets
	dataset_list = [ds.copy() for ds in datasets]
	for ds in dataset_list:
		ds['Options'] = {}
	for spec in params['InputOptions']:
		dataset_list[spec['Idx']]['Options'] = json.loads(spec['Value'])

	# get items
	# only fetch once per unique dataset
	items = {}
	unique_ds_ids = set([ds['ID'] for ds in datasets])
	for ds_id in unique_ds_ids:
		cur_items = requests.get(url+'/datasets/{}/items'.format(ds_id)).json()
		for item in cur_items:
			key = item['Key']
			if key in items:
				items[key][ds_id] = item
			else:
				items[key] = {ds_id: item}
	# only keep keys that exist in all datasets
	for key in list(items.keys()):
		if len(items[key]) != len(unique_ds_ids):
			del items[key]

	keys = list(items.keys())
	random.shuffle(keys)
	num_val = len(keys)//5
	val_keys = keys[0:num_val]
	train_keys = keys[num_val:]

	train_set = MyDataset(dataset_list, train_keys, items)
	val_set = MyDataset(dataset_list, val_keys, items)

	return train_set, val_set

device = torch.device('cuda:0')
#device = torch.device('cpu')

train_set, val_set = get_datasets()
train_loader = torch.utils.data.DataLoader(train_set, batch_size=32, shuffle=True, num_workers=4, collate_fn=train_set.collate_fn)
val_loader = torch.utils.data.DataLoader(val_set, batch_size=32, shuffle=True, num_workers=4, collate_fn=val_set.collate_fn)

example_inputs = train_set.collate_fn([train_set[0]])[0:arch['NumInputs']]
net = model.Net(arch, comps, example_inputs)
net.to(device)
optimizer = torch.optim.Adam(net.parameters(), lr=1e-3)
updated_lr = False

#optimizer = torch.optim.Adam(net.parameters(), lr=1e-3, betas=(0.937, 0.999))

#ckpt = torch.load('/home/ubuntu/skyhookml/yolov3/yolov3.pt')
#state_dict = ckpt['model'].float().state_dict()
#state_dict = {'mlist.0.model.'+k: v for k, v in state_dict.items()}
#state_dict = {k: v for k, v in state_dict.items() if k in net.state_dict() and k not in ['anchor'] and not k.startswith('mlist.0.model.model.28.')}
#net.load_state_dict(state_dict, strict=False)

#for k, v in net.named_parameters():
#	v.requires_grad = k.startswith('mlist.0.model.model.28.')

# number of epochs with no improvement in loss
stop_count = 0
epoch = 0

best_loss = None
while stop_count < 20:
	net.train()
	for inputs in train_loader:
		util.inputs_to_device(inputs, device)
		optimizer.zero_grad()
		loss_dict, _ = net(*inputs[0:arch['NumInputs']], targets=inputs[arch['NumInputs']:])
		loss_dict['loss'].backward()
		optimizer.step()

	val_losses = []
	net.eval()
	for inputs in val_loader:
		util.inputs_to_device(inputs, device)
		loss_dict, _ = net(*inputs[0:arch['NumInputs']], targets=inputs[arch['NumInputs']:])
		val_losses.append({k: v.item() for k, v in loss_dict.items()})
	val_loss_avgs = {}
	for k in val_losses[0].keys():
		val_loss_avgs[k] = numpy.mean([d[k] for d in val_losses])
	val_loss = val_loss_avgs['loss']

	print(val_loss_avgs)
	print('val_loss={}/{}'.format(val_loss, best_loss))

	if best_loss is None or val_loss < best_loss:
		best_loss = val_loss
		stop_count = 0
		torch.save(net.get_save_dict(), 'models/{}.pt'.format(node_id))
	else:
		stop_count += 1
		if not updated_lr and stop_count > 10:
			print('set learning rate lower')
			optimizer = torch.optim.Adam(net.parameters(), lr=1e-4)
			updated_lr = True
			stop_count = 0

	epoch += 1
