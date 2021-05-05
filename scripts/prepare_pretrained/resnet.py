import hashlib
import json
import numpy
import os.path
import sys
import torch
import yaml

sys.path.append('./python/')
import skyhook.pytorch.model as model
import skyhook.pytorch.util as util

mode = sys.argv[1]
out_fname = sys.argv[2]

device = torch.device('cpu')

# get arch, comps
with open('exec_ops/pytorch/archs/resnet.json', 'r') as f:
    arch = json.load(f)
with open('python/skyhook/pytorch/components/resnet.json', 'r') as f:
    comps = {'resnet': {'ID': 'resnet', 'Params': json.load(f)}}

# set mode
comp_params = json.loads(arch['Components'][0].get('Params', '{}'))
comp_params['mode'] = mode
arch['Components'][0]['Params'] = json.dumps(comp_params)

# example inputs
im_data = numpy.zeros((224, 224, 3), dtype='uint8')
int_data = 0
example_inputs = [
    util.collate('image', [util.prepare_input('image', im_data, {}, {})]),
    util.collate('int', [util.prepare_input('int', int_data, {}, {})]),
]
util.inputs_to_device(example_inputs, device)

# example metadata
with open('scripts/prepare_pretrained/imagenet.txt', 'r') as f:
    categories = [line.strip() for line in f.readlines() if line.strip()]
example_metadatas = [{}, {'Categories': categories}]

import skyhook.pytorch.components.resnet
skyhook.pytorch.components.resnet.Pretrain = True

net = model.Net(arch, comps, example_inputs, example_metadatas, device=device)
torch.save(net.get_save_dict(), out_fname)
