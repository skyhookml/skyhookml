import os.path
import skyhook.common as lib
import torch
import yaml

import skyhook.pytorch.components.yolov5_common as yolov5_common

def M(info):
	with yolov5_common.ImportContext() as ctx:
		import utils.general
		import utils.loss
		import models.yolo

		class Yolov5(torch.nn.Module):
			def __init__(self, info):
				super(Yolov5, self).__init__()
				self.infer = info['infer']
				detection_metadata = info['metadatas'][1]
				if detection_metadata and 'Categories' in detection_metadata:
					self.categories = detection_metadata['Categories']
				else:
					self.categories = ['object']
				self.nc = len(self.categories)

				# e.g. 's', 'm', 'l', 'x'
				self.mode = info['params'].get('mode', 'x')

				self.confidence_threshold = info['params'].get('confidence_threshold', 0.1)
				self.iou_threshold = info['params'].get('iou_threshold', 0.5)

				lib.eprint('yolov5: set nc={}, mode={}, conf={}, iou={}'.format(self.nc, self.mode, self.confidence_threshold, self.iou_threshold))

				with open(os.path.join(ctx.expected_path, 'data', 'hyp.scratch.yaml'), 'r') as f:
					hyp = yaml.load(f, Loader=yaml.FullLoader)
				self.model = models.yolo.Model(cfg=os.path.join(ctx.expected_path, 'models', 'yolov5{}.yaml'.format(self.mode)), nc=self.nc, anchors=hyp.get('anchors'))
				self.model.nc = self.nc
				self.model.hyp = hyp
				self.model.gr = 1.0
				self.model.class_weights = torch.ones((self.nc,), dtype=torch.float32)
				self.model.names = self.categories

				# we need to set it onto device early since compute_loss copies the anchors
				self.model.to(info['device'])
				self.compute_loss = utils.loss.ComputeLoss(self.model)

			def forward(self, x, targets=None):
				if targets is not None:
					targets = yolov5_common.process_targets(targets[0])

				d = {}

				if self.training:
					d['pred'] = self.model(x.float()/255.0)
				else:
					inf_out, d['pred'] = self.model(x.float()/255.0)

					if self.infer:
						detections = utils.general.non_max_suppression(inf_out, conf_thres=self.confidence_threshold, iou_thres=self.iou_threshold)
						d['detections'] = yolov5_common.process_outputs((x.shape[3], x.shape[2]), detections, self.categories)

				if targets is not None:
					loss, _ = self.compute_loss(d['pred'], targets)
					d['loss'] = torch.mean(loss)

				return d

		return Yolov5(info)
