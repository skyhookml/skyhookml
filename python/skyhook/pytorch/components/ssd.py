import hashlib
import os.path
import skyhook.common as lib
import sys
import torch
import yaml

class ImportContext(object):
	def __init__(self):
		self.expected_path = os.path.join('.', 'data', 'models', hashlib.sha256(b'https://github.com/qfgaohao/pytorch-ssd.git').hexdigest())

	def __enter__(self):
		# from github.com/ultralytics/yolov3
		sys.path.insert(1, self.expected_path)
		return self

	def __exit__(self, exc_type, exc_value, traceback):
		# reset sys.modules
		for module_name in list(sys.modules.keys()):
			if not hasattr(sys.modules[module_name], '__file__'):
				continue
			fname = sys.modules[module_name].__file__
			if fname is None:
				continue
			if not fname.startswith(self.expected_path):
				continue
			del sys.modules[module_name]
		sys.path.remove(self.expected_path)

def M(info):
	with ImportContext() as ctx:
		from vision.ssd.ssd import MatchPrior
		from vision.ssd.vgg_ssd import create_vgg_ssd
		from vision.ssd.mobilenetv1_ssd import create_mobilenetv1_ssd
		from vision.ssd.mobilenetv1_ssd_lite import create_mobilenetv1_ssd_lite
		from vision.ssd.squeezenet_ssd_lite import create_squeezenet_ssd_lite
		from vision.ssd.mobilenet_v2_ssd_lite import create_mobilenetv2_ssd_lite
		from vision.ssd.mobilenetv3_ssd_lite import create_mobilenetv3_large_ssd_lite, create_mobilenetv3_small_ssd_lite
		from vision.nn.multibox_loss import MultiboxLoss
		from vision.ssd.config import vgg_ssd_config
		from vision.ssd.config import mobilenetv1_ssd_config
		from vision.ssd.config import squeezenet_ssd_config
		from vision.utils import box_utils

		def predict(boxes, scores, config):
			# adapted from predictor.py
			# we need to copy this code because in pytorch-ssd the Predictor wraps entire network rather than being modular
			prob_threshold = 0.01
			candidate_size = 200
			iou_threshold = config.iou_threshold
			sigma = 0.5
			nms_method = None
			top_k = -1

			cpu_device = torch.device('cpu')
			orig_device = boxes.device
			boxes = boxes.to(cpu_device)
			scores = scores.to(cpu_device)
			picked_box_probs = []
			picked_labels = []
			for class_index in range(1, scores.size(1)):
				probs = scores[:, class_index]
				mask = probs > prob_threshold
				probs = probs[mask]
				if probs.size(0) == 0:
					continue
				subset_boxes = boxes[mask, :]
				box_probs = torch.cat([subset_boxes, probs.reshape(-1, 1)], dim=1)
				box_probs = box_utils.nms(box_probs, nms_method,
										  score_threshold=prob_threshold,
										  iou_threshold=iou_threshold,
										  sigma=sigma,
										  top_k=top_k,
										  candidate_size=candidate_size)
				picked_box_probs.append(box_probs)
				picked_labels.extend([class_index] * box_probs.size(0))
			if not picked_box_probs:
				return torch.zeros(0, 4, dtype=torch.float32, device=boxes.device), torch.zeros(0, dtype=torch.float32, device=boxes.device), torch.zeros(0, dtype=torch.float32, device=boxes.device)
			picked_box_probs = torch.cat(picked_box_probs)
			return picked_box_probs[:, :4].to(orig_device), torch.tensor(picked_labels).to(orig_device), picked_box_probs[:, 4].to(orig_device)

		class SSD(torch.nn.Module):
			def __init__(self, info):
				super(SSD, self).__init__()
				self.infer = info['infer']
				detection_metadata = info['metadatas'][1]
				if detection_metadata and 'Categories' in detection_metadata:
					self.categories = detection_metadata['Categories']
				else:
					self.categories = ['object']
				self.num_classes = len(self.categories)+1
				lib.eprint('ssd: set num_classes={}'.format(self.num_classes))

				self.mode = info['params'].get('mode', 'mb2-ssd-lite')
				mb2_width_mult = info['params'].get('mb2_width_mult', 1.0)

				# adapt from train_ssd.py
				if self.mode == 'vgg16-ssd':
					create_net = create_vgg_ssd
					config = vgg_ssd_config
				elif self.mode == 'mb1-ssd':
					create_net = create_mobilenetv1_ssd
					config = mobilenetv1_ssd_config
				elif self.mode == 'mb1-ssd-lite':
					create_net = create_mobilenetv1_ssd_lite
					config = mobilenetv1_ssd_config
				elif self.mode == 'sq-ssd-lite':
					create_net = create_squeezenet_ssd_lite
					config = squeezenet_ssd_config
				elif self.mode == 'mb2-ssd-lite':
					create_net = lambda num, is_test: create_mobilenetv2_ssd_lite(num, width_mult=mb2_width_mult, is_test=is_test)
					config = mobilenetv1_ssd_config
				elif self.mode == 'mb3-large-ssd-lite':
					create_net = lambda num: create_mobilenetv3_large_ssd_lite(num, is_test=is_test)
					config = mobilenetv1_ssd_config
				elif self.mode == 'mb3-small-ssd-lite':
					create_net = lambda num: create_mobilenetv3_small_ssd_lite(num, is_test=is_test)
					config = mobilenetv1_ssd_config

				self.config = config
				self.model = create_net(self.num_classes, is_test=self.infer)
				self.criterion = MultiboxLoss(config.priors, iou_threshold=0.5, neg_pos_ratio=3, center_variance=0.1, size_variance=0.2, device=torch.device('cuda:0'))
				self.match_prior = MatchPrior(config.priors, config.center_variance, config.size_variance, 0.5)
				self.image_mean = torch.tensor(self.config.image_mean, dtype=torch.float32).reshape(1, 3, 1, 1)

			def forward(self, x, targets=None):
				device = x.device
				cpu_device = torch.device('cpu')

				# pre-process image:
				# (1) subtract config.image_mean
				# (2) divide by config.image_std
				x = (x.float() - self.image_mean.to(device=device)) / self.config.image_std

				# pre-process boxes
				if targets is not None:
					target = targets[0]
					counts = target['counts']
					# pytorch-ssd expects class 0 to be background class
					labels = target['detections'][:, 0].long()+1
					boxes = target['detections'][:, 1:5]

					# xyxy -> xywh
					boxes = torch.clamp(boxes, 0, 1)

					start = 0
					target_boxes = []
					target_labels = []
					for count in counts:
						cur_labels = labels[start:start+count]
						cur_boxes = boxes[start:start+count]
						cur_boxes, cur_labels = self.match_prior(cur_boxes.to(device=cpu_device), cur_labels.to(device=cpu_device))
						target_boxes.append(cur_boxes)
						target_labels.append(cur_labels)
						start += count
					target_boxes = torch.stack(target_boxes, dim=0).to(device)
					target_labels = torch.stack(target_labels, dim=0).to(device)

				d = {}

				# apply network
				scores, boxes = self.model(x)

				# compute output detections
				d['detections'] = None
				if not self.training and self.infer:
					counts = []
					dlists = []
					for i in range(boxes.shape[0]):
						cur_boxes, cur_labels, cur_probs = predict(boxes[i], scores[i], self.config)
						counts.append(cur_boxes.shape[0])

						# convert from xywh to xyxy and make it [cls, xyxy, conf]
						dlists.append(torch.stack([
							cur_labels-1,
							cur_boxes[:, 0],
							cur_boxes[:, 1],
							cur_boxes[:, 2],
							cur_boxes[:, 3],
							#cur_boxes[:, 0]-cur_boxes[:, 2]/2,
							#cur_boxes[:, 1]-cur_boxes[:, 3]/2,
							#cur_boxes[:, 0]+cur_boxes[:, 2]/2,
							#cur_boxes[:, 1]+cur_boxes[:, 3]/2,
							cur_probs,
						], dim=1))

					d['detections'] = {
						'counts': torch.tensor(counts, dtype=torch.int32, device=device),
						'detections': torch.cat(dlists, dim=0),
						'categories': self.categories,
					}

				if targets is not None:
					regression_loss, classification_loss =  self.criterion(scores, boxes, target_labels, target_boxes)
					loss = regression_loss + classification_loss
					d['loss'] = loss
					d['regress_loss'] = regression_loss
					d['classify_loss'] = classification_loss

				return d

		return SSD(info)
