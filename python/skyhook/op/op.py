import skyhook.common as lib
import skyhook.io

import io
import json
import requests

class Operator:
	def __init__(self, meta_packet):
		self.inputs = meta_packet['Inputs']
		self.outputs = meta_packet['Outputs']
		self.url = meta_packet['URL']
		self.local_url = 'http://127.0.0.1:{}'.format(meta_packet['Port'])

	def parallelism(self):
		return 1

	def get_tasks(self, raw_items):
		# Default get_tasks corresponds to exec_ops.SimpleTasks.
		groups = {}
		for i, items in enumerate(raw_items['inputs']):
			# Add new keys.
			cur_keys = set()
			for item in items:
				k = item['Key']
				cur_keys.add(k)
				if i == 0:
					groups[k] = [item]
				elif k in groups:
					groups[k].append(item)

			# Remove keys not in this input.
			for k in list(groups.keys()):
				if k not in cur_keys:
					del groups[k]

		tasks = []
		for k, items in groups.items():
			tasks.append({
				'Key': k,
				'Items': {'inputs': [[item] for item in items]},
			})

		return tasks

	def read_item(self, dataset, item):
		resp = requests.post(self.local_url + '/load-data', json=item, stream=True)
		resp.raise_for_status()
		metadata = lib.decode_metadata(dataset, item)
		data = skyhook.io.read_datas(resp.raw, [dataset['DataType']], [metadata])[0]
		return data, metadata

	def write_item(self, dataset, key, data, metadata):
		buf = io.BytesIO()
		skyhook.io.write_json(buf, {
			'Dataset': dataset,
			'Key': key,
			'Metadata': json.dumps(metadata),
		})
		skyhook.io.write_datas(buf, [dataset['DataType']], [data])
		requests.post(self.local_url + '/write-item', data=buf.getvalue()).raise_for_status()

	def apply(self, task):
		raise NotImplementedError
