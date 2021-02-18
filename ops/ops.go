package ops

import (
	_ "../train_ops/pytorch"
	_ "../train_ops/unsupervised_reid"
	_ "../train_ops/yolov3"
	_ "../exec_ops/detection_filter"
	_ "../exec_ops/filter"
	_ "../exec_ops/model"
	_ "../exec_ops/python"
	_ "../exec_ops/reid_tracker"
	_ "../exec_ops/render"
	_ "../exec_ops/simple_tracker"
	_ "../exec_ops/video_sample"
)
