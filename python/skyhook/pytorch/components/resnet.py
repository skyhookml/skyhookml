import skyhook.common as lib
import torch
import torchvision
import torchvision.models.resnet as resnet

# Provide flag to pass pretrained=True to resnet model function.
# This makes it easy for us to develop an external script that prepares
# pre-trained parameters in a SkyhookML file dataset.
Pretrain = False

# hyperparameter constants
Means = [0.485, 0.456, 0.406]
Std = [0.229, 0.224, 0.225]

class Resnet(torch.nn.Module):
	def __init__(self, info):
		super(Resnet, self).__init__()
		example_inputs = info['example_inputs']

		# detect number of classes from int metadata
		num_classes = 1000
		if len(info['metadatas']) >= 2:
			int_metadata = info['metadatas'][1]
			if int_metadata and 'Categories' in int_metadata:
				num_classes = len(int_metadata['Categories'])

		# configurable options
		mode = info['params'].get('mode', 'resnet34')
		num_classes = info['params'].get('num_classes', num_classes)
		lib.eprint('resnet set mode={} num_classes={}'.format(mode, num_classes))

		model_func = None
		if mode == 'resnet18':
			model_func = resnet.resnet18
		elif mode == 'resnet34':
			model_func = resnet.resnet34
		elif mode == 'resnet50':
			model_func = resnet.resnet50
		elif mode == 'resnet101':
			model_func = resnet.resnet101
		elif mode == 'resnet152':
			model_func = resnet.resnet152
		self.model = model_func(pretrained=Pretrain, progress=Pretrain, num_classes=num_classes)

		self.normalize = torchvision.transforms.Normalize(mean=Means, std=Std)
		self.ce = torch.nn.CrossEntropyLoss()
		self.softmax = torch.nn.Softmax(dim=1)

	def forward(self, x, targets=None):
		x = x.float()/255.0
		x = self.normalize(x)
		scores = self.model(x)

		d = {
			'scores': scores,
			'probs': self.softmax(scores),
			'cls': torch.argmax(scores, dim=1),
		}

		if targets is not None and len(targets) > 0:
			d['loss'] = torch.mean(self.ce(scores, targets[0]))

		return d

def M(info):
	return Resnet(info)
