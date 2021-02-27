import utils from './utils.js';
import AnnotateGenericUI from './annotate-generic-ui.js';

export default AnnotateGenericUI({
	data: function() {
		return {
			detections: null,
			metadata: null,
			nextTrackID: null,

			// whether the current item has existing labels
			itemHasExisting: false,
		};
	},
	on_update: function() {
		this.detections = [];
		for(let i = 0; i < this.numFrames; i++) {
			this.detections.push([]);
		}
		this.nextTrackID = 1;
		this.itemHasExisting = false;

		// get actual detections from source
		// but don't overwrite if we already loaded detections from on_item_data
		let srcDataset = this.annoset.Inputs[1];
		let params = {
			format: 'json',
			t: new Date().getTime(),
		};
		utils.request(this, 'GET', '/datasets/'+srcDataset.ID+'/items/'+this.response.Key+'/get?format=json', params, (data) => {
			if(!this.itemHasExisting) {
				this.detections = data;
			}
		});
		utils.request(this, 'GET', '/datasets/'+srcDataset.ID+'/items/'+this.response.Key+'/get?format=meta', params, (data) => {
			if(!this.itemHasExisting) {
				this.metadata = data;
			}
		});
	},
	on_item_data: function(data, metadata) {
		if(data.length == 0) {
			return;
		}
		this.detections = data;
		this.metadata = metadata;
		this.itemHasExisting = true;

		// next track ID should be one higher than the maximum
		this.detections.forEach((detection) => {
			if(!detection.TrackID) {
				return;
			}
			if(detection.TrackID >= this.nextTrackID) {
				this.nextTrackID = detection.TrackID + 1;
			}
		});

		// update if we already rendered image
		if(this.imageDims != null) {
			this.render();
		}
	},
	on_image_loaded: function() {
		Vue.nextTick(() => {
			this.render();
		});
	},
	getAnnotateData: function() {
		return [this.detections, this.metadata];
	},
	methods: {
		render: function() {
			let stage = new Konva.Stage({
				container: this.$refs.layer,
				width: this.imageDims.Width,
				height: this.imageDims.Height,
			});
			let layer = new Konva.Layer();

			// draw detections without track ID in yellow
			// and remaining detections in other colors
			this.detections[this.frameIdx].forEach((detection) => {
				let stroke = detection.TrackID ? 'red' : 'yellow';
				let rect = new Konva.Rect({
					x: detection.Left,
					y: detection.Top,
					width: detection.Right-detection.Left,
					height: detection.Bottom-detection.Top,
					stroke: stroke,
					strokeWidth: 3,
				});

				rect.on('click', (e) => {
					// only for already-labeled detections, delete the label on click
					if(!detection.TrackID) {
						return;
					}
					e.cancelBubble = true;
					delete detection.TrackID;
					rect.stroke('yellow');
					layer.draw();
				});

				layer.add(rect);
			});

			stage.on('click', (e) => {
				let pos = stage.getPointerPosition();
				// find closest unlabeled detection, using top-left of bounding box
				let bestDetection = null;
				let bestDistance = null;
				this.detections[this.frameIdx].forEach((detection) => {
					if(detection.TrackID) {
						return;
					}
					let dx = detection.Left - pos.x;
					let dy = detection.Right - pos.y;
					let distance = dx*dx+dy*dy;
					if(bestDetection === null || distance < bestDistance) {
						bestDetection = detection;
						bestDistance = distance;
					}
				});
				if(bestDetection) {
					bestDetection.TrackID = this.nextTrackID;
				}
				// advance to next frame with at least one unlabeled detection
				// if no such frames, increment track counter
				//   and then find first frame with unlabeled detection if any
				let frameIdx = this.unlabeledFrameAfter(this.frameIdx+1);
				if(frameIdx >= 0) {
					this.getFrame(frameIdx);
				} else {
					this.endTrack();
				}
			});

			stage.add(layer);
			layer.draw();
		},
		unlabeledFrameAfter: function(start) {
			for(let frameIdx = start; frameIdx < this.detections.length; frameIdx++) {
				let ok = false;
				this.detections[frameIdx].forEach((detection) => {
					if(!detection.TrackID) {
						ok = true;
					}
				});
				if(ok) {
					return frameIdx;
				}
			}
			return -1;
		},
		endTrack: function() {
			this.nextTrackID++;
			let frameIdx = this.unlabeledFrameAfter(0);
			if(frameIdx >= 0) {
				this.getFrame(frameIdx);
			} else {
				this.getFrame(this.frameIdx);
			}
		},
	},
	template: {
		im_above: `
<div class="form-row align-items-center">
	<div class="col-auto">
		Labeling Track {{ this.nextTrackID }}
	</div>
	<div class="col-auto" v-if="response != null">
		<button type="button" class="btn btn-primary" v-on:click="endTrack">End Track</button>
	</div>
</div>
		`,
		im_after: `
<div
	v-if="imageDims != null"
	ref="layer"
	:style="{
		width: imageDims.Width+'px',
		height: imageDims.Height+'px',
	}"
	>
</div>
		`,
	},
});
