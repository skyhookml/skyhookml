import json
import torchvision
import torch.optim
import torch.nn

import sys
def eprint(s):
	sys.stderr.write(str(s) + "\n")
	sys.stderr.flush()

class Net(torch.nn.Module):
	def __init__(self, arch, comps, params, example_inputs):
		super(Net, self).__init__()

		self.arch = arch
		self.comps = comps
		self.params = params
		self.example_inputs = example_inputs

		# collect modules
		self.mlist = torch.nn.ModuleList()
		example_layers = {}
		for comp_idx, comp_spec in enumerate(self.arch['Components']):
			comp = self.comps[comp_spec['ID']]

			# get example inputs for this component
			cur_inputs = []
			for inp_idx, inp_spec in enumerate(comp_spec['Inputs']):
				if inp_spec['Type'] == 'layer':
					layer = example_layers[inp_spec['ComponentIdx']][inp_spec['Layer']]
					cur_inputs.append(layer)
				else:
					cur_inputs.append(example_inputs[inp_spec['DatasetIdx']])

			# extract params
			cur_params = None
			if comp_spec['Params']:
				cur_params = json.loads(comp_spec['Params'])

			locals = {}
			exec(comp['Code'], None, locals)
			m = locals['M'](cur_params, cur_inputs)
			self.mlist.append(m)
			example_layers[comp_idx] = m(*cur_inputs)

	def forward(self, *inputs, targets=None):
		layers = {}

		def get_input_or_target(idx):
			if idx < len(inputs):
				return inputs[idx]
			else:
				return targets[idx-len(inputs)]

		def get_layer(spec):
			return layers[spec['ComponentIdx']][spec['Layer']]

		# collect layers
		for comp_idx, comp_spec in enumerate(self.arch['Components']):
			comp = self.comps[comp_spec['ID']]

			# inputs
			cur_inputs = []
			for inp_idx, inp_spec in enumerate(comp_spec['Inputs']):
				if inp_spec['Type'] == 'layer':
					layer = get_layer(inp_spec)
					cur_inputs.append(layer)
				else:
					cur_inputs.append(inputs[inp_spec['DatasetIdx']])

			# targets
			if targets is None:
				cur_targets = None
			else:
				cur_targets = []
				for inp_idx, inp_spec in enumerate(comp_spec['Targets']):
					if inp_spec['Type'] == 'layer':
						layer = get_layer(inp_spec)
						cur_targets.append(layer)
					else:
						layer = get_input_or_target(inp_spec['DatasetIdx'])
						cur_targets.append(layer)

			cur_outputs = self.mlist[comp_idx](*cur_inputs, targets=cur_targets)
			layers[comp_idx] = cur_outputs

		# collect outputs
		outputs = []
		for out_spec in self.params['OutputDatasets']:
			layer = get_layer(out_spec)
			outputs.append(layer)

		if targets is None:
			return tuple(outputs)

		# compute loss
		loss_dict = {'loss': 0}
		for i, loss_spec in enumerate(self.arch['Losses']):
			layer = get_layer(loss_spec)
			loss_dict['loss{}_{}'.format(i, loss_spec['Layer'])] = layer
			loss_dict['loss'] += layer * loss_spec['Weight']

		return (loss_dict, outputs)

	def get_save_dict(self):
		return {
			'model': self.state_dict(),
			'example_inputs': self.example_inputs,
		}
