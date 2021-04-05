import utils from './utils.js';
import ExecCropResize from './exec-edit-cropresize.js';
import ExecDetectionFilter from './exec-edit-detection_filter.js';
import ExecResample from './exec-edit-resample.js';
import ExecSegmentationMask from './exec-edit-segmentation_mask.js';
import ExecSimpleTracker from './exec-edit-simple_tracker.js';
import ExecReidTracker from './exec-edit-reid_tracker.js';
import ExecPython from './exec-edit-python.js';
import ExecPytorchTrain from './exec-edit-pytorch_train.js';
import ExecPytorchInfer from './exec-edit-pytorch_infer.js';
import ExecPytorchYolov3Train from './exec-edit-pytorch_yolov3_train.js';
import ExecPytorchYolov3Infer from './exec-edit-pytorch_yolov3_infer.js';
import ExecPytorchYolov5Train from './exec-edit-pytorch_yolov5_train.js';
import ExecPytorchYolov5Infer from './exec-edit-pytorch_yolov5_infer.js';
import ExecYolov3Train from './exec-edit-yolov3_train.js';
import ExecYolov3Infer from './exec-edit-yolov3_infer.js';
import ExecUnsupervisedReid from './exec-edit-pytorch_train.js';
import ExecVideoSample from './exec-edit-video_sample.js';

let components = {
	'cropresize': ExecCropResize,
	'detection_filter': ExecDetectionFilter,
	'resample': ExecResample,
	'segmentation_mask': ExecSegmentationMask,
	'simple_tracker': ExecSimpleTracker,
	'reid_tracker': ExecReidTracker,
	'python': ExecPython,
	'pytorch_train': ExecPytorchTrain,
	'pytorch_infer': ExecPytorchInfer,
	'pytorch_yolov3_train': ExecPytorchYolov3Train,
	'pytorch_yolov3_infer': ExecPytorchYolov3Infer,
	'pytorch_yolov5_train': ExecPytorchYolov5Train,
	'pytorch_yolov5_infer': ExecPytorchYolov5Infer,
	'yolov3_train': ExecYolov3Train,
	'yolov3_infer': ExecYolov3Infer,
	'unsupervised_reid': ExecUnsupervisedReid,
	'video_sample': ExecVideoSample,
};

export default {
	components: components,
	data: function() {
		return {
			node: null,
			components: Object.keys(components),
		};
	},
	created: function() {
		utils.request(this, 'GET', '/exec-nodes/'+this.$route.params.nodeid, null, (node) => {
			this.node = node;
		});
	},
	template: `
<div class="el-high">
	<template v-if="node">
		<template v-if="components.includes(node.Op)">
			<component v-if="node" v-bind:is="node.Op" v-bind:node="node"></component>
		</template>
		<template v-else>
			<p>This node does not have parameters.</p>
		</template>
	</template>
</div>
	`,
};
