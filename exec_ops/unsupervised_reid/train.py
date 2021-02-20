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

sys.path.append('./exec_ops/pytorch/')
import model
import util

MIN_PADDING = 4
CROP_SIZE = 64
MATCH_LENGTHS = [8, 16, 32, 64]

node_id = int(sys.argv[1])
url = sys.argv[2]
params_arg = sys.argv[3]
arch_arg = sys.argv[4]
comps_arg = sys.argv[5]
datasets_arg = sys.argv[6]
matches_path = sys.argv[7]

params = json.loads(params_arg)
arch = json.loads(arch_arg)
comps = json.loads(comps_arg)
datasets = json.loads(datasets_arg)

arch = arch['Params']
comps = {int(comp_id): comp['Params'] for comp_id, comp in comps.items()}
datasets = {int(ds_id): dataset for ds_id, dataset in datasets.items()}

def clip(x, lo, hi):
	if x > hi:
		return hi
	elif x < lo:
		return lo
	else:
		return x

class MyDataset(torch.utils.data.Dataset):
	def __init__(self, datasets, keys, items, is_val=False):
		self.datasets = datasets
		self.keys = keys
		self.items = items

		# datasets[0] should be video, datasets[1] should be detections
		# and matches are in matches_paths
		# so here we want to load all the detections, with 64x64 images
		video_ds = self.datasets[0]
		detection_ds = self.datasets[1]
		self.data = {}
		self.options = []
		for key in self.keys:
			print('loading from', key)
			cur_items = self.items[key]
			video_item, detection_item = cur_items[video_ds['ID']], cur_items[detection_ds['ID']]
			detections = lib.load_item(detection_ds, detection_item)
			orig_dims = json.loads(detection_item['Metadata'])['CanvasDims']
			for (frame_idx, im) in enumerate(lib.load_item(video_ds, video_item)):
				if frame_idx >= len(detections):
					continue

				if orig_dims[0] == 0:
					orig_dims = (im.shape[1], im.shape[0])

				for d in detections[frame_idx]:
					left = clip(d['Left']*im.shape[1]//orig_dims[0], 0, im.shape[1]-MIN_PADDING)
					right = clip(d['Right']*im.shape[1]//orig_dims[0], left+MIN_PADDING, im.shape[1])
					top = clip(d['Top']*im.shape[0]//orig_dims[1], 0, im.shape[0]-MIN_PADDING)
					bottom = clip(d['Bottom']*im.shape[0]//orig_dims[1], top+MIN_PADDING, im.shape[0])
					crop = im[top:bottom, left:right, :]

					resize_factor = min([float(CROP_SIZE) / crop.shape[0], float(CROP_SIZE) / crop.shape[1]])
					resize_shape = [int(crop.shape[0] * resize_factor), int(crop.shape[1] * resize_factor)]
					crop = skimage.transform.resize(crop, resize_shape, preserve_range=True).astype('uint8')

					fix_crop = numpy.zeros((CROP_SIZE, CROP_SIZE, 3), dtype='uint8')
					fix_crop[0:crop.shape[0], 0:crop.shape[1], :] = crop

					d['im'] = fix_crop

			with open(os.path.join(matches_path, key+'.json'), 'r') as f:
				raw_matches = json.load(f)
			matches = {}
			for frame_idx, detection_idx, match_length, next_idx in raw_matches:
				k = (frame_idx, detection_idx, match_length)
				if k not in matches:
					matches[k] = []
				matches[k].append(next_idx)

			self.data[key] = (detections, matches)
			for frame_idx in range(len(detections)):
				for match_length in MATCH_LENGTHS:
					if frame_idx+match_length >= len(detections):
						continue
					if len(detections[frame_idx]) == 0 or len(detections[frame_idx+match_length]) == 0:
						continue
					self.options.append((key, frame_idx, match_length))

		if is_val and len(self.options) > 256:
			self.options = random.sample(self.options, 256)

	def __len__(self):
		return len(self.options)

	def __getitem__(self, idx):
		if torch.is_tensor(idx):
			idx = idx.tolist()

		key, frame_idx, match_length = self.options[idx]
		detections, matches = self.data[key]

		prev_detections = detections[frame_idx]
		next_detections = detections[frame_idx+match_length]

		prev_images = [d['im'] for d in prev_detections]
		next_images = [numpy.zeros((CROP_SIZE, CROP_SIZE, 3), dtype='uint8')] + [d['im'] for d in next_detections]
		mask = numpy.zeros((len(prev_images), len(next_images)), dtype='float32')
		for prev_idx in range(len(prev_detections)):
			k = (frame_idx, prev_idx, match_length)
			if k not in matches:
				continue
			for next_idx in matches[k]:
				mask[prev_idx, next_idx+1] = 1

		return [
			torch.stack([torch.from_numpy(im.transpose(2, 0, 1)) for im in prev_images], dim=0),
			torch.stack([torch.from_numpy(im.transpose(2, 0, 1)) for im in next_images], dim=0),
			torch.from_numpy(mask),
		]

def get_datasets():
	# get flat dataset list
	dataset_list = []
	for ds_spec in params['InputDatasets']:
		dataset = datasets[ds_spec['ID']].copy()
		dataset['Options'] = {}
		if ds_spec['Options']:
			dataset['Options'] = json.loads(ds_spec['Options'])
		dataset_list.append(dataset)

	# get items
	items = {}
	for dataset in datasets.values():
		ds_id = dataset['ID']
		cur_items = requests.get(url+'/datasets/{}/items'.format(ds_id)).json()
		for item in cur_items:
			key = item['Key']
			if key in items:
				items[key][ds_id] = item
			else:
				items[key] = {ds_id: item}
	# only keep keys that exist in all datasets
	for key in list(items.keys()):
		if len(items[key]) != len(datasets):
			del items[key]

	keys = list(items.keys())
	random.shuffle(keys)
	num_val = len(keys)//5
	val_keys = keys[0:num_val]
	train_keys = keys[num_val:]

	train_set = MyDataset(dataset_list, train_keys, items)
	val_set = MyDataset(dataset_list, val_keys, items, is_val=True)

	return train_set, val_set

device = torch.device('cuda:0')
#device = torch.device('cpu')

train_set, val_set = get_datasets()
train_loader = torch.utils.data.DataLoader(train_set, batch_size=None, shuffle=True, num_workers=4)
val_loader = torch.utils.data.DataLoader(val_set, batch_size=None, shuffle=True, num_workers=4)

example_inputs = train_set[0]
net = model.Net(arch, comps, params, example_inputs)
net.to(device)
optimizer = torch.optim.Adam(net.parameters(), lr=1e-3)
updated_lr = False

# number of epochs with no improvement in loss
stop_count = 0
epoch = 0

best_loss = None
while stop_count < 20:
	net.train()
	for i, inputs in enumerate(train_loader):
		util.inputs_to_device(inputs, device)
		optimizer.zero_grad()
		loss_dict, _ = net(*inputs[0:arch['NumInputs']], targets=inputs[arch['NumInputs']:])
		loss_dict['loss'].backward()
		optimizer.step()
		if i >= 1024:
			break

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

	epoch += 1
