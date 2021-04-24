import numpy
import random

def clip(x, lo, hi):
	if x < lo:
		return lo
	elif x > hi:
		return hi
	else:
		return x

class Crop(object):
	def __init__(self, params, data_types):
		# width and height are fractions between 0 and 1
		# they indicate the relative width/height of the crop to the original dimensions
		def parse_fraction(s):
			if '/' in s:
				parts = s.split('/')
				return (int(parts[0]), int(parts[1]))
			else:
				return float(s), 1
		self.Width = parse_fraction(params['Width'])
		self.Height = parse_fraction(params['Height'])
		self.data_types = data_types

		# this augmentation should be applied in the Dataset
		self.pre_torch = True

	def forward(self, batch):
		# pick offsets to crop
		# we use same offsets for entire batch
		xoff = random.random() * (1 - self.Width[0]/self.Width[1])
		yoff = random.random() * (1 - self.Height[0]/self.Height[1])

		# transform coordinates in image to coordinates in cropped image
		def coord_transform(x, y):
			x -= xoff
			y -= yoff
			x *= self.Width[1]/self.Width[0]
			y *= self.Height[1]/self.Height[0]
			return x, y

		for i, inputs in enumerate(batch):
			if self.data_types[i] in ('image', 'video', 'array'):
				width, height = inputs[0].shape[2], inputs[0].shape[1]
				target_width = int(width * self.Width[0]) // self.Width[1]
				target_height = int(height * self.Height[0]) // self.Height[1]
				sx = int(xoff * width)
				sy = int(yoff * height)
				ex = sx + target_width
				ey = sy + target_height

				ninputs = []
				for input in inputs:
					input = input[:, sy:ey, sx:ex]
					ninputs.append(input)
				batch[i] = ninputs

			elif self.data_types[i] == 'shape':
				# shapes can be arbitrary polygons
				# so we don't really want to support it
				raise Exception('crop is not supported on shape types')

			elif self.data_types[i] == 'detection':
				# we clip detection bboxes inside the crop rectangle
				# but we also remove ones that are entirely outside the crop rectangle
				ninputs = []
				for input in inputs:
					ndetections = []
					for d in input['detections']:
						cls, sx, sy, ex, ey = d
						sx, sy = coord_transform(sx, sy)
						ex, ey = coord_transform(ex, ey)
						if ex < 0 or sx >= 1 or ey < 0 or sy >= 1:
							continue
						sx = clip(sx, 0, 1)
						sy = clip(sy, 0, 1)
						ex = clip(ex, 0, 1)
						ey = clip(ey, 0, 1)
						ndetections.append([cls, sx, sy, ex, ey])
					ndetections = numpy.array(ndetections, dtype='float32')
					ninputs.append({
						'counts': ndetections.shape[0],
						'detections': ndetections,
					})
				batch[i] = ninputs

			# we assume that other data types like int, floats, etc. don't need to be cropped


		return batch
