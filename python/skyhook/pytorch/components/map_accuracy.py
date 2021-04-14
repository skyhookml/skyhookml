import numpy
import torch

# Mean average precision metric for object detection.
# Detection format: (cls, xyxy, conf)

# Group detections by image, and move to numpy.
def group_by_image(raw_detections):
	counts = raw_detections['counts'].detach().cpu().numpy()
	raw_dlist = raw_detections['detections'].detach().cpu().numpy()
	prefix_sum = 0
	dlists = []
	for count in counts:
		dlists.append(raw_dlist[prefix_sum:prefix_sum+count])
		prefix_sum += count
	return dlists

# Group detections by category IDs.
# But only keep categories with at least one ground truth detection.
def group_by_category(pred, gt):
	categories = {}
	for d in gt:
		category_id = int(d[0])
		if category_id not in categories:
			categories[category_id] = ([], [])
		categories[category_id][1].append(d[1:5])
	for d in pred:
		category_id = int(d[0])
		if category_id not in categories:
			continue
		categories[category_id][0].append(d[1:6])
	return categories

# Return intersection-over-union between boxes.
# Returns IOU, where IOU[i, j] is IOU between pred[i] and gt[j]
def get_iou(pred, gt):
	def box_area(box):
		return (box[:, 2] - box[:, 0]) * (box[:, 3] - box[:, 1])

	area1 = box_area(pred)
	area2 = box_area(gt)

	intersect_area = (numpy.minimum(pred[:, None, 2:4], gt[:, 2:4]) - numpy.maximum(pred[:, None, 0:2], gt[:, 0:2]))
	intersect_area = numpy.maximum(intersect_area, 0)
	intersect_area = numpy.prod(intersect_area, axis=2)
	union_area = area1[:, None] + area2 - intersect_area
	return intersect_area / union_area

def compute_ap(recall, precision):
	# Append sentinel values to beginning and end
	mrec = numpy.concatenate(([0.], recall, [recall[-1] + 0.01]))
	mpre = numpy.concatenate(([1.], precision, [0.]))

	# Compute the precision envelope
	mpre = numpy.flip(numpy.maximum.accumulate(numpy.flip(mpre)))

	# Integrate area under curve with interp method
	x = numpy.linspace(0, 1, 101)  # 101-point interp (COCO)
	ap = numpy.trapz(numpy.interp(x, mrec, mpre), x)  # integrate

	return ap

class MapAccuracy(torch.nn.Module):
	def __init__(self):
		super(MapAccuracy, self).__init__()
		self.iou_threshold = 0.5

	def forward(self, detections, targets=None):
		if targets is None or self.training:
			return {}

		orig_device = detections['counts'].device
		detections = detections
		targets = targets[0]

		ap_scores = []
		# Loop over detections in each category and in each image.
		detections = group_by_image(detections)
		targets = group_by_image(targets)
		for image_idx in range(len(detections)):
			by_categories = group_by_category(detections[image_idx], targets[image_idx])
			for cls in by_categories.keys():
				pred, gt = by_categories[cls]

				if len(pred) == 0:
					ap_scores.append(0)
					continue

				# sort by confidence
				pred.sort(key=lambda d: d[4])
				pred = numpy.stack(pred, axis=0)
				gt = numpy.stack(gt, axis=0)

				# get iou matrix
				iou_mat = get_iou(pred, gt)

				# match predicted detections with ground truth detections
				tp = numpy.zeros((len(pred),), dtype='bool')
				fp = numpy.zeros((len(pred),), dtype='bool')
				gt_seen = numpy.zeros((len(gt),), dtype='bool')
				for idx1, d1 in enumerate(pred):
					best_idx = iou_mat[idx1, :].argmax()
					best_iou = iou_mat[idx1, best_idx]
					if best_iou > self.iou_threshold and not gt_seen[best_idx]:
						gt_seen[best_idx] = True
						tp[idx1] = True
					else:
						fp[idx1] = True

				# get precision and recall curves
				tp_curve = numpy.cumsum(tp.astype('int32'))
				fp_curve = numpy.cumsum(fp.astype('int32'))
				recall = tp_curve / len(gt)
				precision = tp_curve / (fp_curve + tp_curve)

				# compute ap
				ap_score = compute_ap(recall, precision)
				ap_scores.append(ap_score)

		map_score = numpy.mean(ap_scores)
		return {
		 	'score': torch.as_tensor(map_score, dtype=torch.float32, device=orig_device),
		}

def M(info):
	return MapAccuracy()
