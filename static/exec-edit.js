import utils from './utils.js';
import CropResize from './exec-edit/cropresize.js';
import DetectionFilter from './exec-edit/detection_filter.js';
import GeoImageToImage from './exec-edit/geoimage_to_image.js';
import MakeGeoImage from './exec-edit/make_geoimage.js';
import Resample from './exec-edit/resample.js';
import SegmentationMask from './exec-edit/segmentation_mask.js';
import SimpleTracker from './exec-edit/simple_tracker.js';
import ReidTracker from './exec-edit/reid_tracker.js';
import Python from './exec-edit/python.js';
import PytorchTrain from './exec-edit/pytorch_train.js';
import PytorchInfer from './exec-edit/pytorch_infer.js';
import PytorchResnetTrain from './exec-edit/pytorch_resnet_train.js';
import PytorchResnetInfer from './exec-edit/pytorch_resnet_infer.js';
import PytorchSsdTrain from './exec-edit/pytorch_ssd_train.js';
import PytorchSsdInfer from './exec-edit/pytorch_ssd_infer.js';
import PytorchUnetTrain from './exec-edit/pytorch_unet_train.js';
import PytorchUnetInfer from './exec-edit/pytorch_unet_infer.js';
import PytorchYolov3Train from './exec-edit/pytorch_yolov3_train.js';
import PytorchYolov3Infer from './exec-edit/pytorch_yolov3_infer.js';
import PytorchYolov5Train from './exec-edit/pytorch_yolov5_train.js';
import PytorchYolov5Infer from './exec-edit/pytorch_yolov5_infer.js';
import Sample from './exec-edit/sample.js';
import SpatialFlowPartition from './exec-edit/spatialflow_partition.js';
import Yolov3Train from './exec-edit/yolov3_train.js';
import Yolov3Infer from './exec-edit/yolov3_infer.js';
import UnsupervisedReid from './exec-edit/unsupervised_reid.js';
import VideoSample from './exec-edit/video_sample.js';

let components = {
	'cropresize': CropResize,
	'detection_filter': DetectionFilter,
	'geoimage_to_image': GeoImageToImage,
	'make_geoimage': MakeGeoImage,
	'resample': Resample,
	'segmentation_mask': SegmentationMask,
	'simple_tracker': SimpleTracker,
	'reid_tracker': ReidTracker,
	'python': Python,
	'pythonv2': Python,
	'pytorch_train': PytorchTrain,
	'pytorch_infer': PytorchInfer,
	'pytorch_resnet_train': PytorchResnetTrain,
	'pytorch_resnet_infer': PytorchResnetInfer,
	'pytorch_ssd_train': PytorchSsdTrain,
	'pytorch_ssd_infer': PytorchSsdInfer,
	'pytorch_unet_train': PytorchUnetTrain,
	'pytorch_unet_infer': PytorchUnetInfer,
	'pytorch_yolov3_train': PytorchYolov3Train,
	'pytorch_yolov3_infer': PytorchYolov3Infer,
	'pytorch_yolov5_train': PytorchYolov5Train,
	'pytorch_yolov5_infer': PytorchYolov5Infer,
	'yolov3_train': Yolov3Train,
	'yolov3_infer': Yolov3Infer,
	'spatialflow_partition': SpatialFlowPartition,
	'sample': Sample,
	'unsupervised_reid': UnsupervisedReid,
	'video_sample': VideoSample,
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

			this.$store.commit('setRouteData', {
				node: this.node,
			});
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
