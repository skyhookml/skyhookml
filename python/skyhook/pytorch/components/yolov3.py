import os.path
import skyhook.common as lib
import torch
import yaml

import skyhook.pytorch.components.yolov3_common as yolov3_common

def M(info):
	with yolov3_common.ImportContext() as ctx:
		import utils.general
		import utils.loss
		import models.yolo

		class Yolov3(torch.nn.Module):
			def __init__(self, info):
				super(Yolov3, self).__init__()
				self.infer = info['infer']
				detection_metadata = info['metadatas'][1]
				if detection_metadata and 'Categories' in detection_metadata:
					self.categories = detection_metadata['Categories']
				else:
					self.categories = ['object']
				self.nc = len(self.categories)

				# e.g. 'yolov3', 'yolov3-tiny', 'yolov3-spp'
				self.mode = info['params'].get('mode', 'yolov3')

				if self.infer:
					default_confidence_threshold = 0.1
				else:
					default_confidence_threshold = 0.01
				self.confidence_threshold = info['params'].get('confidence_threshold', default_confidence_threshold)
				self.iou_threshold = info['params'].get('iou_threshold', 0.5)

				lib.eprint('yolov3: set nc={}, mode={}, conf={}, iou={}'.format(self.nc, self.mode, self.confidence_threshold, self.iou_threshold))

				self.model = models.yolo.Model(cfg=os.path.join(ctx.expected_path, 'models', '{}.yaml'.format(self.mode)), nc=self.nc)
				self.model.nc = self.nc
				with open(os.path.join(ctx.expected_path, 'data', 'hyp.scratch.yaml'), 'r') as f:
					hyp = yaml.load(f, Loader=yaml.FullLoader)
				self.model.hyp = hyp
				self.model.gr = 1.0
				self.model.class_weights = torch.ones((self.nc,), dtype=torch.float32)
				self.model.names = self.categories

			def forward(self, x, targets=None):
				if targets is not None:
					targets = yolov3_common.process_targets(targets[0])

				d = {}

				if self.training:
					d['pred'] = self.model(x.float()/255.0)
					d['detections'] = None
				else:
					inf_out, d['pred'] = self.model(x.float()/255.0)
					detections = utils.general.non_max_suppression(inf_out, self.confidence_threshold, self.iou_threshold)
					d['detections'] = yolov3_common.process_outputs((x.shape[3], x.shape[2]), detections, self.categories)

				if targets is not None:
					loss, _ = utils.loss.compute_loss(d['pred'], targets, self.model)
					d['loss'] = torch.mean(loss)

				return d

		return Yolov3(info)
