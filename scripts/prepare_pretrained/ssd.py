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
in_fname = sys.argv[2]
out_fname = sys.argv[3]

device = torch.device('cpu')
ssd_path = os.path.join('.', 'data', 'models', hashlib.sha256(b'https://github.com/qfgaohao/pytorch-ssd.git').hexdigest())

# get arch, comps
with open('exec_ops/pytorch/archs/ssd.json', 'r') as f:
    arch = json.load(f)
with open('python/skyhook/pytorch/components/ssd.json', 'r') as f:
    comps = {'ssd': {'ID': 'ssd', 'Params': json.load(f)}}

# set mode
comp_params = json.loads(arch['Components'][0].get('Params', '{}'))
comp_params['mode'] = mode
arch['Components'][0]['Params'] = json.dumps(comp_params)

# example inputs
im_data = numpy.zeros((300, 300, 3), dtype='uint8')
detection_data = [{'Left': 100, 'Right': 100, 'Top': 100, 'Bottom': 100}]
# Need to repeat the inputs twice because SSD requires batch_size>1 for normalization.
example_inputs = [
    util.collate('image', 2*[util.prepare_input('image', im_data, {}, {})]),
    util.collate('detection', 2*[util.prepare_input('detection', detection_data, {'CanvasDims': [300, 300]}, {})]),
]
util.inputs_to_device(example_inputs, device)

# example metadata
categories = [
    'aeroplane', 'bicycle', 'bird', 'boat', 'bottle', 'bus', 'car', 'cat',
    'chair', 'cow', 'diningtable', 'dog', 'horse', 'motorbike', 'person',
    'pottedplant', 'sheep', 'sofa', 'train', 'tvmonitor',
]
example_metadatas = [{}, {'Categories': categories}]

net = model.Net(arch, comps, example_inputs, example_metadatas, device=device)

sys.path.append(ssd_path)
orig_dict = torch.load(in_fname)
state_dict = {}
for k, v in orig_dict.items():
    state_dict['mlist.0.model.'+k] = v
net.load_state_dict(state_dict)

torch.save(net.get_save_dict(), out_fname)
