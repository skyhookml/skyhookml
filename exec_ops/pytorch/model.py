import hashlib
import importlib.util
import json
import torchvision
import torch.optim
import torch.nn
import sys

def eprint(s):
	sys.stderr.write(str(s) + "\n")
	sys.stderr.flush()

class Net(torch.nn.Module):
	def __init__(self, arch, comps, example_inputs, example_metadatas, output_datasets=None):
		super(Net, self).__init__()

		self.arch = arch
		self.comps = comps
		self.example_inputs = example_inputs
		self.example_metadatas = example_metadatas
		self.output_datasets = output_datasets

		# collect modules
		self.mlist = torch.nn.ModuleList()
		example_layers = {}
		for comp_idx, comp_spec in enumerate(self.arch['Components']):
			comp = self.comps[comp_spec['ID']]

			# get example inputs/metadatas for this component
			# this includes both inputs and targets
			cur_inputs = []
			cur_metadatas = []
			num_inputs = len(comp_spec['Inputs'])
			num_targets = len(comp_spec['Targets'])

			for inp_spec in comp_spec['Inputs'] + comp_spec['Targets']:
				if inp_spec['Type'] == 'layer':
					layer = example_layers[inp_spec['ComponentIdx']][inp_spec['Layer']]
					cur_inputs.append(layer)
					cur_metadatas.append(None)
				else:
					cur_inputs.append(example_inputs[inp_spec['DatasetIdx']])
					cur_metadatas.append(example_metadatas[inp_spec['DatasetIdx']])

			# extract params
			cur_params = None
			if comp_spec['Params']:
				cur_params = json.loads(comp_spec['Params'])

			cur_info = {
				'params': cur_params,
				'example_inputs': cur_inputs,
				'metadatas': cur_metadatas,
			}

			module_spec = comp.get('Module')
			if module_spec.get('BuiltInModule', None):
				module = importlib.import_module('skyhook_components.' + module_spec['BuiltInModule'], package='skyhook_components')
				m = module.M(cur_info)
			elif module_spec.get('RepositoryModule', None):
				repo = module_spec['Repository']
				repo_id = repo['URL']
				if repo.get('Commit', None):
					repo_id += '@' + repo['Commit']
				expected_path = os.path.join('.', 'models', hashlib.sha256(repo_id.encode()).hexdigest(), module_spec['RepositoryModule']+'.py')
				module_name = 'comp{}.{}'.format(comp['ID'], module_spec['RepositoryModule'])
				spec = importlib.util.spec_from_file_location(module_name, expected_path)
				module = importlib.util.module_from_spec(spec)
				m = module.M(cur_info)
			elif module_spec.get('Code', None):
				locals = {}
				exec(module_spec['Code'], None, locals)
				m = locals['M'](cur_info)
			else:
				raise Exception('invalid module {}: none of BuiltInModule, RepositoryModule, or Code are set'.format(comp['ID']))

			self.mlist.append(m)

			# get example layers by running forward pass with only the inputs (no targets)
			example_layers[comp_idx] = m(*cur_inputs[0:num_inputs])

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
		if self.output_datasets is None:
			outputs = [layers]
		else:
			outputs = []
			for out_spec in self.output_datasets:
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
			'arch': self.arch,
			'comps': self.comps,
			'example_inputs': self.example_inputs,
			'example_metadatas': self.example_metadatas,
		}
