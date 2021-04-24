import random
import torch
import torchvision

class RandomResize(object):
	def __init__(self, params, data_types):
		self.min_width = params['MinWidth']
		self.min_height = params['MinHeight']
		self.max_width = params['MaxWidth']
		self.max_height = params['MaxHeight']
		self.keep_ratio = params['KeepRatio']
		self.multiple = params['Multiple']
		self.data_types = data_types
		self.pre_torch = False

	def forward(self, batch):
		for i, x in enumerate(batch):
			if self.data_types[i] not in ('image', 'video', 'array'):
				# we only need to resize images
				# other data types like object detections are represented in a way that doesn't depend on scale
				continue

			if self.keep_ratio:
				width = random.randint(self.min_width, self.max_width)
				factor = width / x.shape[3]
				height = int(factor * x.shape[2])
			else:
				width = random.randint(self.min_width, self.max_width)
				height = random.randint(self.min_height, self.max_height)

			if self.multiple > 1:
				width = (width + self.multiple - 1) // self.multiple * self.multiple
				height = (height + self.multiple - 1) // self.multiple * self.multiple

			batch[i] = torchvision.transforms.functional.resize(x, size=[height, width])
		return batch
