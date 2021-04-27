from skyhook.op.op import Operator

import skyhook.common as lib
import skyhook.io

import io
import requests

class PerFrameOperator(Operator):
	def __init__(self, meta_packet):
		super(PerFrameOperator, self).__init__(meta_packet)
		# Function must be set after initialization.
		self.f = None

	def apply(self, task):
		# Use SynchronizedReader to read the input items chunk-by-chunk.
		items = [item_list[0] for item_list in task['Items']['inputs']]
		rd_resp = requests.post(self.local_url + '/synchronized-reader', json=items, stream=True)
		rd_resp.raise_for_status()

		# Define generator function that will run self.f on each element of sequence data.
		def gen():
			# First we need to yield encoded metadata.
			metas = []
			for ds in self.outputs:
				metas.append({
					'Dataset': ds,
					'Key': task['Key'],
				})
			buf = io.BytesIO()
			skyhook.io.write_json(buf, metas)
			yield buf.getvalue()

			while True:
				dtypes = [ds['DataType'] for ds in self.inputs]
				# Read a chunk.
				try:
					datas = skyhook.io.read_datas(rd_resp.raw, dtypes)
				except EOFError:
					break

				# Collect output chunk by running self.f on each element.
				input_len = lib.data_len(dtypes[0], datas[0])
				outputs = []
				for i in range(input_len):
					cur_inputs = [lib.data_index(dtypes[ds_idx], data, i) for ds_idx, data in enumerate(datas)]
					cur_outputs = self.f(*cur_inputs)
					if not isinstance(cur_outputs, tuple):
						cur_outputs = (cur_outputs,)
					outputs.append(cur_outputs)

				# Stack the outputs, encode them, and yield the bytes.
				out_dtypes = [ds['DataType'] for ds in self.outputs]
				out_datas = []
				for i, t in enumerate(out_dtypes):
					data = lib.data_stack(t, [output[i] for output in outputs])
					out_datas.append(data)
				buf = io.BytesIO()
				skyhook.io.write_datas(buf, out_dtypes, out_datas)
				yield buf.getvalue()

		# Make the request.
		requests.post(self.local_url + '/build', data=gen()).raise_for_status()

def per_frame(f):
	def wrap(meta_packet):
		op = PerFrameOperator(meta_packet)
		op.f = f
		return op
	return wrap
