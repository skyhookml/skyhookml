import keras.backend, keras.layers, keras.models
import tensorflow as tf

def get_model(archs, params):
	layers = {}
	inputs = []
	outputs = []

	for i in range(len(archs)):
		arch = archs[i]
		arch_meta = params['Archs'][i]
		for j, inp in enumerate(arch['Params']['Inputs']):
			src = arch_meta['Inputs'][j]
			if src == 'input':
				print(inp[0], '=', inp[1])
				layers[inp[0]] = eval(inp[1], None, layers)
				inputs.append(layers[inp[0]])
			else:
				layers[inp[0]] = layers[src]
		for item in arch['Params']['Arch']:
			print(item[0], '=', item[1])
			layers[item[0]] = eval(item[1], None, layers)
		for name in arch_meta['Outputs']:
			outputs.append(layers[name])

	model = keras.models.Model(inputs=inputs, outputs=outputs)
	return model, layers
