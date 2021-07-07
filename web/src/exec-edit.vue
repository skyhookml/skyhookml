<template>
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
</template>

<script>
import utils from './utils.js';
import CropResize from './exec-edit/cropresize.vue';
import DetectionFilter from './exec-edit/detection_filter.vue';
import ExtractPolygons from './exec-edit/extract_polygons.vue';
import GeoImageToImage from './exec-edit/geoimage_to_image.vue';
import MakeGeoImage from './exec-edit/make_geoimage.vue';
import Python from './exec-edit/python.vue';
import PytorchTrain from './exec-edit/pytorch_train.js';
import PytorchInfer from './exec-edit/pytorch_infer.vue';
import PytorchResnetTrain from './exec-edit/pytorch_resnet_train.js';
import PytorchResnetInfer from './exec-edit/pytorch_resnet_infer.vue';
import PytorchSsdTrain from './exec-edit/pytorch_ssd_train.js';
import PytorchSsdInfer from './exec-edit/pytorch_ssd_infer.vue';
import PytorchUnetTrain from './exec-edit/pytorch_unet_train.js';
import PytorchUnetInfer from './exec-edit/pytorch_unet_infer.vue';
import PytorchYolov3Train from './exec-edit/pytorch_yolov3_train.js';
import PytorchYolov3Infer from './exec-edit/pytorch_yolov3_infer.vue';
import PytorchYolov5Train from './exec-edit/pytorch_yolov5_train.js';
import PytorchYolov5Infer from './exec-edit/pytorch_yolov5_infer.vue';
import Sample from './exec-edit/sample.vue';
import SegmentationMask from './exec-edit/segmentation_mask.vue';
import SimpleTracker from './exec-edit/simple_tracker.vue';
import Split from './exec-edit/split.vue';
import ReidTracker from './exec-edit/reid_tracker.vue';
import Resample from './exec-edit/resample.vue';
import Yolov3Train from './exec-edit/yolov3_train.vue';
import Yolov3Infer from './exec-edit/yolov3_infer.vue';
import UnsupervisedReid from './exec-edit/unsupervised_reid.js';
import VideoSample from './exec-edit/video_sample.vue';

let components = {
	'cropresize': CropResize,
	'detection_filter': DetectionFilter,
	'extract_polygons': ExtractPolygons,
	'geoimage_to_image': GeoImageToImage,
	'make_geoimage': MakeGeoImage,
	'python': Python,
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
	'sample': Sample,
	'segmentation_mask': SegmentationMask,
	'simple_tracker': SimpleTracker,
	'split': Split,
	'reid_tracker': ReidTracker,
	'resample': Resample,
	'yolov3_train': Yolov3Train,
	'yolov3_infer': Yolov3Infer,
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
};
</script>
