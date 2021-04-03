import skyhook.common as lib
import torch

class ClsHead(torch.nn.Module):
	def __init__(self, info):
		super(ClsHead, self).__init__()
		example_inputs = info['example_inputs']
		kernel = 3
		self.ch = example_inputs[0].shape[1]
		padding = kernel//2

		# detect number of classes from int metadata
		int_metadata = info['metadatas'][1]
		if int_metadata and 'Categories' in int_metadata:
			num_classes = len(int_metadata['Categories'])
		else:
			num_classes = 2

		# configurable options
		layers = info['params'].get('layers', 1)
		features = info['params'].get('features', 128)
		num_classes = info['params'].get('num_classes', num_classes)
		lib.eprint('cls set num_classes={}'.format(num_classes))

		self.relu = torch.nn.ReLU()

		convs = []
		side = min(example_inputs[0].shape[2], example_inputs[0].shape[3])
		for i in range(layers):
			if i == 0:
				in_ch = self.ch
			else:
				in_ch = features

			# set stride 2 unless resolution is already low
			if side <= 4:
				stride = 1
			else:
				stride = 2

			conv = torch.nn.Conv2d(in_ch, features, kernel, padding=(padding, padding), stride=stride)
			convs.append(conv)
		self.convs = torch.nn.ModuleList(convs)

		self.fc = torch.nn.Linear(features, num_classes)
		self.ce = torch.nn.CrossEntropyLoss()

	def forward(self, x, targets=None):
		for conv in self.convs:
			x = self.relu(conv(x))
		x = torch.amax(x, dim=[2, 3])
		x = self.fc(x)

		d = {
			'pre_out': x,
			'out': torch.argmax(x, dim=1),
		}

		if targets is not None:
			d['loss'] = torch.mean(self.ce(x, targets[0]))

		return d

def M(info):
	return ClsHead(info)
