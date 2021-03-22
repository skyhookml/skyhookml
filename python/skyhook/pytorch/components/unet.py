import torch

# adapted from https://github.com/usuyama/pytorch-unet/

class UNet(torch.nn.Module):
	def __init__(self, params, example_inputs):
		super(UNet, self).__init__()
		input_channels = example_inputs[0].shape[1]

		if params is None:
			params = {}

		kernel = params.get('kernel', 3)
		channels_list = params.get('channels_list', [64, 128, 256, 512])
		num_classes = params.get('num_classes', 2)
		# reduction of input scale when computing output
		# for example, 8 means we reduce width/height both by 8x
		# must be a power of 2
		scale = params.get('scale', 1)

		padding = kernel//2

		def double_conv(in_channels, out_channels):
			return torch.nn.Sequential(
				torch.nn.Conv2d(in_channels, out_channels, kernel, padding=padding),
				torch.nn.ReLU(inplace=True),
				torch.nn.Conv2d(out_channels, out_channels, kernel, padding=padding),
				torch.nn.ReLU(inplace=True)
			)

		self.maxpool = torch.nn.MaxPool2d(2)
		self.upsample = torch.nn.Upsample(scale_factor=2, mode='bilinear', align_corners=True)
		self.ce = torch.nn.CrossEntropyLoss()
		self.softmax = torch.nn.Softmax(dim=1)

		down_layers = []
		up_layers = []
		up_channels = []

		for i in range(len(channels_list)-1):
			layer = double_conv(channels_list[i], channels_list[i+1])
			down_layers.append(layer)

			if 2**i < scale:
				continue

			if i == 0:
				layer_in_channels = channels_list[i+1]
			else:
				layer_in_channels = channels_list[i+1] + channels_list[i]
			layer = double_conv(layer_in_channels, channels_list[i])
			up_layers = [layer] + up_layers
			up_channels = [channels_list[i]] + up_channels

		self.down_layers = torch.nn.ModuleList(down_layers)
		self.up_layers = torch.nn.ModuleList(up_layers)
		self.conv_first = double_conv(input_channels, channels_list[0])
		self.conv_last = torch.nn.Conv2d(up_channels[-1], num_classes, 1)

	def forward(self, x, targets=None):
		x = x.float()/255.0
		outputs = {}
		down_outputs = []

		x = self.conv_first(x)

		for i, layer in enumerate(self.down_layers):
			x = self.maxpool(x)
			x = layer(x)
			outputs['down{}'.format(i)] = x
			down_outputs.append(x)

		for i, layer in enumerate(self.up_layers):
			x = self.upsample(x)
			# we want to walk i back from the end to create skip connections
			# -(i+2) because +1 for starting from end and another +1 for not using the very last encoding layer
			if i != len(self.up_layers)-1:
				x = torch.cat([x, down_outputs[-(i+2)]], dim=1)
			x = layer(x)
			outputs['up{}'.format(i)] = x

		x = self.conv_last(x)

		outputs['scores'] = x
		outputs['probs'] = self.softmax(x)
		outputs['classes'] = torch.argmax(x, dim=1, keepdim=True)

		if targets is not None:
			outputs['loss'] = torch.mean(self.ce(x, targets[0][:, 0, :, :].long()))

		return outputs

def M(info):
	return UNet(info['params'], info['example_inputs'])
