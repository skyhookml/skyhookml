import random
import torch
import torchvision

class Flip(object):
	def __init__(self, params, data_types):
		self.mode = params['Mode']
		self.data_types = data_types
		self.pre_torch = False

	def forward(self, batch):
		horizontal_flip = False
		vertical_flip = False
		if self.mode == 'both' or self.mode == 'horizontal':
			horizontal_flip = random.random() < 0.5
		if self.mode == 'both' or self.mode == 'vertical':
			vertical_flip = random.random() < 0.5

		for i, inputs in enumerate(batch):
			if self.data_types[i] == 'image':
				flip_dims = []
				if horizontal_flip:
					flip_dims.append(3)
				if vertical_flip:
					flip_dims.append(2)

				if flip_dims:
					batch[i] = torch.flip(inputs, flip_dims)

			elif self.data_types[i] == 'detection':
				cls = inputs['detections'][:, 0]
				sx = inputs['detections'][:, 1]
				sy = inputs['detections'][:, 2]
				ex = inputs['detections'][:, 3]
				ey = inputs['detections'][:, 4]

				# if we flip the coordinates, the smaller one becomes the bigger one
				# so we need to swap the start/end coordinates too here, if we flip
				if horizontal_flip:
					sx, ex = 1-ex, 1-sx
				if vertical_flip:
					sy, ey = 1-ey, 1-sy

				inputs['detections'] = torch.stack([cls, sx, sy, ex, ey], dim=1)

			elif self.data_types[i] == 'shape':
				px = inputs['points'][:, 0]
				py = inputs['points'][:, 1]
				if horizonal_flip:
					px = 1-px
				if vertical_flip:
					py = 1-py

				inputs['points'] = torch.stack([px, py], dim=1)

		return batch
