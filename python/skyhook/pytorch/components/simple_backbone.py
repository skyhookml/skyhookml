import torch

class SimpleBackbone(torch.nn.Module):
	def __init__(self, params, example_inputs):
		super(SimpleBackbone, self).__init__()
		input_channels = example_inputs[0].shape[1]

		if params is None:
			params = {}

		kernel = params.get('kernel', 3)
		channels_list = params.get('channels_list', [32, 64, 128, 256, 512])
		strides_list = params.get('strides', [2]*len(channels_list))
		batch_norm = params.get('batch_norm', True)

		padding = kernel//2
		channels_list = [input_channels] + channels_list

		layers = []
		for i in range(len(channels_list)-1):
			cur = []
			conv = torch.nn.Conv2d(channels_list[i], channels_list[i+1], kernel, padding=(padding, padding), stride=strides_list[i])
			cur.append(conv)
			if batch_norm:
				bn = torch.nn.BatchNorm2d(channels_list[i+1], eps=1e-3, momentum=0.03)
				cur.append(bn)
			cur.append(torch.nn.ReLU())

			layers.append(torch.nn.Sequential(*cur))

		self.layers = torch.nn.ModuleList(layers)

	def forward(self, x, targets=None):
		x = x.float()/255.0
		outputs = {}
		for i, layer in enumerate(self.layers):
			x = layer(x)
			outputs['layer{}'.format(i)] = x
		return outputs

def M(info):
	return SimpleBackbone(info['params'], info['example_inputs'])
