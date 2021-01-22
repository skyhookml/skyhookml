package ops

import (
	_ "../train_ops/pytorch"
	_ "../train_ops/yolov3"
	_ "../exec_ops/filter"
	_ "../exec_ops/model"
	_ "../exec_ops/python"
	_ "../exec_ops/render"
	_ "../exec_ops/video_sample"
)
