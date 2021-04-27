from skyhook.op.op import Operator

import skyhook.common as lib
import skyhook.io

import io
import requests

class AllDecorateOperator(Operator):
	def __init__(self, meta_packet):
		super(AllDecorateOperator, self).__init__(meta_packet)
		# Function must be set after initialization.
		self.f = None

	def apply(self, task):
		# Use LoadData to fetch datas one by one.
		items = [item_list[0] for item_list in task['Items']['inputs']]
		datas = []
		for i, item in enumerate(items):
			resp = requests.post(self.local_url + '/load-data', json=item, stream=True)
			resp.raise_for_status()
			data = skyhook.io.read_datas(resp.raw, [self.inputs[i]['DataType']])[0]
			datas.append(data)

		# Run the user-defined function.
		outputs = self.f(*datas)
		if not isinstance(outputs, tuple):
			outputs = (outputs,)

		# Write each output item.
		for i, data in enumerate(outputs):
			buf = io.BytesIO()
			skyhook.io.write_json(buf, {
				'Dataset': self.outputs[i],
				'Key': task['Key'],
			})
			skyhook.io.write_datas(buf, [self.outputs[i]['DataType']], [data])
			requests.post(self.local_url + '/write-item', data=buf.getvalue()).raise_for_status()

def all_decorate(f):
	def wrap(meta_packet):
		op = AllDecorateOperator(meta_packet)
		op.f = f
		return op
	return wrap
