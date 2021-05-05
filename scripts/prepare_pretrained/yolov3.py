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
yolo_path = os.path.join('.', 'data', 'models', hashlib.sha256(b'https://github.com/ultralytics/yolov3.git').hexdigest())

# get arch, comps
with open('exec_ops/pytorch/archs/yolov3.json', 'r') as f:
    arch = json.load(f)
with open('python/skyhook/pytorch/components/yolov3.json', 'r') as f:
    comps = {'yolov3': {'ID': 'yolov3', 'Params': json.load(f)}}

# set mode
comp_params = json.loads(arch['Components'][0].get('Params', '{}'))
comp_params['mode'] = mode
arch['Components'][0]['Params'] = json.dumps(comp_params)

# example inputs
im_data = numpy.zeros((416, 416, 3), dtype='uint8')
example_inputs = [
    util.collate('image', [util.prepare_input('image', im_data, {}, {})]),
    util.collate('detection', [util.prepare_input('detection', [], {'CanvasDims': [416, 416]}, {})]),
]
util.inputs_to_device(example_inputs, device)

# example metadata
with open(os.path.join(yolo_path, 'data', 'coco.yaml'), 'r') as f:
    d = yaml.load(f, Loader=yaml.FullLoader)
    categories = d['names']
example_metadatas = [{}, {'Categories': categories}]

net = model.Net(arch, comps, example_inputs, example_metadatas, device=device)

sys.path.append(yolo_path)
orig_dict = torch.load(in_fname)['model'].state_dict()
state_dict = {}
for k, v in orig_dict.items():
    state_dict['mlist.0.model.'+k] = v
net.load_state_dict(state_dict)

torch.save(net.get_save_dict(), out_fname)
