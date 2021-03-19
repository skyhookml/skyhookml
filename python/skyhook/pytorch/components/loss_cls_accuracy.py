import torch

class ClsAccuracy(torch.nn.Module):
	def __init__(self):
		super(ClsAccuracy, self).__init__()

	def forward(self, x, targets=None):
		if targets is None:
			return {}

		# x is [batch, C] probability distribution with C classes
		# targets is [batch] ints indicating the class labels
		outputs = torch.argmax(x, dim=1)
		matches = torch.eq(outputs, targets[0])
		accuracy = torch.mean(matches.to(torch.float32))

		return {
			'accuracy': accuracy,
		}

def M(info):
	return ClsAccuracy()
