import skyhook.common as lib
import torch

class Reid(torch.nn.Module):
	def __init__(self, info):
		super(Reid, self).__init__()

		conv_layers = [
			torch.nn.Conv2d(3, 32, 4, padding=(1, 1), stride=2), # 32x32x32
			torch.nn.Conv2d(32, 64, 4, padding=(1, 1), stride=2), # 16x16x64
			torch.nn.Conv2d(64, 64, 4, padding=(1, 1), stride=2), # 8x8x64
			torch.nn.Conv2d(64, 64, 4, padding=(1, 1), stride=2), # 4x4x64
			torch.nn.Conv2d(64, 64, 4, padding=(1, 1), stride=2), # 2x2x64
			torch.nn.Conv2d(64, 64, 4, padding=(1, 1), stride=2), # 1x1x64
		]
		self.conv_layers = torch.nn.ModuleList(conv_layers)

		match_layers = [
			torch.nn.Linear(128, 256),
			torch.nn.Linear(256, 256),
			torch.nn.Linear(256, 256),
			torch.nn.Linear(256, 1),
		]
		self.match_layers = torch.nn.ModuleList(match_layers)

		self.relu = torch.nn.ReLU()
		self.row_softmax = torch.nn.Softmax(dim=1)
		self.col_softmax = torch.nn.Softmax(dim=0)

	def forward(self, prev_images, next_images, targets=None):
		prev_images = prev_images.float()/255.0
		next_images = next_images.float()/255.0

		def get_features(x):
			for layer in self.conv_layers[:-1]:
				x = layer(x)
				x = self.relu(x)
			x = self.conv_layers[-1](x)
			return x[:, :, 0, 0]

		def get_scores(x):
			for layer in self.match_layers[:-1]:
				x = layer(x)
				x = self.relu(x)
			x = self.match_layers[-1](x)
			return x

		prev_count = prev_images.shape[0]
		next_count = next_images.shape[0]

		prev_features = get_features(prev_images)
		next_features = get_features(next_images)

		pairs = torch.cat([
			prev_features.reshape(prev_count, 1, 64).repeat([1, next_count, 1]),
			next_features.reshape(1, next_count, 64).repeat([prev_count, 1, 1]),
		], dim=2)
		pairs_flat = pairs.reshape(-1, 128)
		scores_flat = get_scores(pairs_flat)
		scores = scores_flat.reshape((prev_count, next_count))
		probs = torch.minimum(
			self.row_softmax(scores),
			self.col_softmax(scores),
		)

		d = {
			'scores': scores,
			'probs': probs,
		}

		if targets:
			mask = targets[0]
			product = probs * probs * mask
			d['loss'] = -torch.log(product.sum(dim=1) + 1e-8).mean()

		return d

def M(info):
	return Reid(info)
