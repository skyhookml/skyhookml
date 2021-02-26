import os.path
import hashlib
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
	import models.yolo

	class Yolov3(torch.nn.Module):
		def __init__(self):
			super(Yolov3, self).__init__()
			self.model = models.yolo.Model(cfg=os.path.join(expected_path, 'models', 'yolov3.yaml'), nc=1)
			self.model.nc = 1
			with open(os.path.join(expected_path, 'data', 'hyp.scratch.yaml'), 'r') as f:
				hyp = yaml.load(f, Loader=yaml.FullLoader)
			self.model.hyp = hyp
			self.model.gr = 1.0
			self.model.class_weights = torch.ones((1,), dtype=torch.float32)
			self.model.names = ['item']

		def forward(self, x, targets=None):
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

			d = {}

			if self.training:
				d['pred'] = self.model(x.float()/255.0)
				d['detections'] = None
			else:
				inf_out, d['pred'] = self.model(x.float()/255.0)

				conf_thresh = 0.1
				iou_thresh = 0.5
				d['detections'] = utils.general.non_max_suppression(inf_out, conf_thresh, iou_thresh)

			if boxes is not None:
				loss, _ = utils.loss.compute_loss(d['pred'], boxes, self.model)
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

	return Yolov3()
