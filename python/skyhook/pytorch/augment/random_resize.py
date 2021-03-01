import random
import torch
import torchvision

class RandomResize(object):
	def __init__(self, params, data_types):
		self.min_width = params['MinWidth']
		self.min_height = params['MinHeight']
		self.max_width = params['MaxWidth']
		self.max_height = params['MaxHeight']
		self.data_types = data_types

	def forward(self, batch):
		for i, x in enumerate(batch):
			if self.data_types[i] != 'image':
				# we only need to resize images
				# other data types like object detections are represented in a way that doesn't depend on scale
				continue

			width = random.randint(self.min_width, self.max_width)
			height = random.randint(self.min_height, self.max_height)
			batch[i] = torchvision.transforms.functional.resize(x, size=[width, height])
