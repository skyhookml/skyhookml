import json
import random
import requests
import torch

import skyhook.common as lib
import skyhook.pytorch.util as util

class Dataset(torch.utils.data.Dataset):
	'''
	Default dataset for training torch models in skyhook.
	- We retrieve tuples of items with the same key across the input datasets.
	- Each tuple is one element of the Dataset.
	- Batches are Python lists wrapping collated data at each index in the tuple.
	'''

	def __init__(self, datasets, keys, items):
		self.datasets = datasets
		self.keys = keys
		self.items = items

		# data augmentation steps
		self.augments = []

		# Extract metadata for each item.
		self.metadatas = {}
		for key in self.keys:
			cur = []
			for dataset in self.datasets:
				item = items[dataset['ID']]
				metadata = lib.decode_metadata(dataset, item)
				cur.append(metadata)
			self.metadatas[key] = cur

	def __len__(self):
		return len(self.keys)

	def __getitem__(self, idx):
		if torch.is_tensor(idx):
			idx = idx.tolist()

		key = self.keys[idx]
		items = self.items[key]
		metadatas = self.metadatas[key]

		inputs = []
		for i, dataset in enumerate(self.datasets):
			item = items[dataset['ID']]
			data = util.read_input(dataset, item)
			data = util.prepare_input(dataset['DataType'], data, metadatas[i], dataset['Options'])
			inputs.append(data)

		return inputs

	def get_metadatas(self, idx):
		'''
		Returns the metadata for the items at the specified index.
		'''
		key = self.keys[idx]
		return self.metadatas[key]

	def get_datatypes(self):
		'''
		Returns the skyhook datatypes that we will provide.
		'''
		l = []
		for dataset in self.datasets:
			l.append(dataset['DataType'])
		return l

	def set_augments(self, augments):
		self.augments = augments

	def collate_fn(self, batch):
		inputs = list(zip(*batch))

		for augment in self.augments:
			inputs = augment.forward(inputs)

		for i, dataset in enumerate(self.datasets):
			inputs[i] = util.collate(dataset['DataType'], inputs[i])

		return inputs

def get_datasets(url, datasets, params, train_keys, val_keys):
	# add options to datasets
	dataset_list = [ds.copy() for ds in datasets]
	for idx, ds in enumerate(dataset_list):
		ds['Options'] = params['InputOptions'][idx]

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

	if not train_keys or not val_keys:
		keys = list(items.keys())
		random.shuffle(keys)
		num_val = len(keys)*params['ValPercent']//100
		val_keys = keys[0:num_val]
		train_keys = keys[num_val:]
		print('split items into {} for training and {} for validation'.format(len(train_keys), len(val_keys)))
	else:
		print('using given splits, with {} for training and {} for validation'.format(len(train_keys), len(val_keys)))

	train_set = Dataset(dataset_list, train_keys, items)
	val_set = Dataset(dataset_list, val_keys, items)

	return train_set, val_set
