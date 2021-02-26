import os.path
import hashlib
import math
import skyhook_pylib as lib
import sys
import torch
import yaml

def M(params, example_inputs):
	# from github.com/ultralytics/yolov3
	expected_path = os.path.join('.', 'models', hashlib.sha256(b'https://github.com/ultralytics/yolov3.git').hexdigest())
	sys.path.insert(1, expected_path)
	import utils.general
	import utils.loss
	import utils.torch_utils
	import utils.autoanchor
	import models.yolo

	class Yolov3(torch.nn.Module):
		def __init__(self, params, example_inputs):
			super(Yolov3, self).__init__()
			self.nc = 1

			assert example_inputs[0].shape[2] == 2*example_inputs[1].shape[2]
			assert example_inputs[1].shape[2] == 2*example_inputs[2].shape[2]

			yolo_yaml = '''
nc: 80
depth_multiple: 1.0
width_multiple: 1.0

anchors:
  - [10,13, 16,30, 33,23]  # P3/8
  - [30,61, 62,45, 59,119]  # P4/16
  - [116,90, 156,198, 373,326]  # P5/32

backbone: []

head:
  [[-1, 1, Bottleneck, [1024, False]],
   [-1, 1, Conv, [512, [1, 1]]],
   [-1, 1, Conv, [1024, 3, 1]],
   [-1, 1, Conv, [512, 1, 1]],
   [-1, 1, Conv, [1024, 3, 1]],  # 7 (P5/32-large)

   [-2, 1, Conv, [256, 1, 1]],
   [-1, 1, nn.Upsample, [None, 2, 'nearest']],
   [[-1, 1], 1, Concat, [1]],  # cat backbone P4
   [-1, 1, Bottleneck, [512, False]],
   [-1, 1, Bottleneck, [512, False]],
   [-1, 1, Conv, [256, 1, 1]],
   [-1, 1, Conv, [512, 3, 1]],  # 14 (P4/16-medium)

   [-2, 1, Conv, [128, 1, 1]],
   [-1, 1, nn.Upsample, [None, 2, 'nearest']],
   [[-1, 0], 1, Concat, [1]],  # cat backbone P3
   [-1, 1, Bottleneck, [256, False]],
   [-1, 2, Bottleneck, [256, False]],  # 19 (P3/8-small)

   [[19, 14, 7], 1, Detect, [nc, anchors]],   # Detect(P3, P4, P5)
  ]
'''
			yolo_cfg = yaml.load(yolo_yaml, Loader=yaml.FullLoader)
			yolo_cfg['nc'] = self.nc
			input_channels = [x.shape[1] for x in example_inputs]
			self.model, _ = models.yolo.parse_model(yolo_cfg, [3]+input_channels)

			# attached hyperparameters (from train.py)
			with open(os.path.join(expected_path, 'data/hyp.scratch.yaml'), 'r') as f:
				hyp = yaml.load(f, Loader=yaml.FullLoader)
			self.hyp = hyp
			self.gr = 1.0
			self.class_weights = torch.ones((1,), dtype=torch.float32)
			self.names = ['item']

			# Build strides, anchors
			m = self.model[-1]
			s = 128  # 2x min stride
			m.stride = torch.tensor([8.0, 16.0, 32.0])  # forward
			m.anchors /= m.stride.view(-1, 1, 1)
			utils.autoanchor.check_anchor_order(m)
			self.stride = m.stride
			self._initialize_biases()  # only run once

			utils.torch_utils.initialize_weights(self)


		def _initialize_biases(self, cf=None):  # initialize biases into Detect(), cf is class frequency
			# https://arxiv.org/abs/1708.02002 section 3.3
			# cf = torch.bincount(torch.tensor(np.concatenate(dataset.labels, 0)[:, 0]).long(), minlength=nc) + 1.
			m = self.model[-1]  # Detect() module
			for mi, s in zip(m.m, m.stride):  # from
				b = mi.bias.view(m.na, -1)  # conv.bias(255) to (3,85)
				b[:, 4] += math.log(8 / (640 / s) ** 2)  # obj (8 objects per 640 image)
				b[:, 5:] += math.log(0.6 / (m.nc - 0.99)) if cf is None else torch.log(cf / cf.sum())  # cls
				mi.bias = torch.nn.Parameter(b.view(-1), requires_grad=True)

		def forward(self, x1, x2, x3, targets=None):
			boxes = None
			if targets is not None:
				# first extract detection counts per image in the batch, and the boxes
				if len(targets[0]) == 3:
					# shape type
					counts, _, points = targets[0]
					boxes = points.reshape(-1, 4)
					# need to make sure that first point is smaller than second point
					boxes = torch.stack([
						torch.minimum(boxes[:, 0], boxes[:, 2]),
						torch.minimum(boxes[:, 1], boxes[:, 3]),
						torch.maximum(boxes[:, 0], boxes[:, 2]),
						torch.maximum(boxes[:, 1], boxes[:, 3]),
					], dim=1)
				elif len(targets[0]) == 2:
					# detection type
					counts, boxes = targets[0]

				# xyxy -> xywh
				boxes = torch.stack([
					(boxes[:, 0] + boxes[:, 2]) / 2,
					(boxes[:, 1] + boxes[:, 3]) / 2,
					boxes[:, 2] - boxes[:, 0],
					boxes[:, 3] - boxes[:, 1],
				], dim=1)

				# output: list of bbox with first column indicating image index
				indices = torch.repeat_interleave(
					torch.arange(counts.shape[0], dtype=torch.int32, device=counts.device).float(),
					counts.long()
				).reshape(-1, 1)
				cls_labels = torch.zeros(indices.shape, dtype=torch.float32, device=counts.device)
				boxes = torch.cat([indices, cls_labels, boxes], dim=1)

			y = [x1, x2, x3]
			x = x3
			for m in self.model:
				if m.f != -1:  # if not from previous layer
					x = y[m.f] if isinstance(m.f, int) else [x if j == -1 else y[j] for j in m.f]  # from earlier layers

				x = m(x)  # run
				y.append(x)

			d = {}

			if self.training:
				d['pred'] = x
				d['detections'] = None
			else:
				inf_out, d['pred'] = x

				conf_thresh = 0.1
				iou_thresh = 0.5
				d['detections'] = utils.general.non_max_suppression(inf_out, conf_thresh, iou_thresh)

			if boxes is not None:
				loss, _ = utils.loss.compute_loss(d['pred'], boxes, self)
				d['loss'] = torch.mean(loss)

			return d

	# reset sys.modules
	for module_name in list(sys.modules.keys()):
		if not hasattr(sys.modules[module_name], '__file__'):
			continue
		fname = sys.modules[module_name].__file__
		if fname is None:
			continue
		if not fname.startswith(expected_path):
			continue
		#lib.eprint('clearing {}'.format(module_name))
		del sys.modules[module_name]
	sys.path.remove(expected_path)

	return Yolov3(params, example_inputs)
