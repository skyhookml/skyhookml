from skyhook.op.op import Operator

import skyhook.common as lib
import skyhook.io

import requests

class AllDecorateOperator(Operator):
	def __init__(self, meta_packet):
		super(AllDecorateOperator, self).__init__(meta_packet)
		# Function must be set after initialization.
		self.f = None

	def apply(self, task):
		# Use LoadData to fetch datas one by one.
		# We combine it with metadata to create the input arguments.
		items = [item_list[0] for item_list in task['Items']['inputs']]
		args = []
		for i, item in enumerate(items):
			data, metadata = self.read_item(self.inputs[i], item)
			args.append({
				'Data': data,
				'Metadata': metadata,
			})

		# Run the user-defined function.
		outputs = self.f(*args)
		if not isinstance(outputs, tuple):
			outputs = (outputs,)

		# Write each output item.
		for i, data in enumerate(outputs):
			if isinstance(data, dict) and 'Data' in data:
				data, metadata = data['Data'], data['Metadata']
			else:
				metadata = {}

			self.write_item(self.outputs[i], task['Key'], data, metadata)

def all_decorate(f):
	def wrap(meta_packet):
		op = AllDecorateOperator(meta_packet)
		op.f = f
		return op
	return wrap
