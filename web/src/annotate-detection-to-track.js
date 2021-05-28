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

			resizeObserver: null,
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
		let srcDataset = this.annoset.InputDatasets[1];
		let params = {
			format: 'json',
			t: new Date().getTime(),
		};
		utils.request(this, 'GET', '/datasets/'+srcDataset.ID+'/items/'+this.response.Key+'/get?format=json', params, (data) => {
			if(!this.itemHasExisting) {
				this.detections = data;
				this.updateIfAlreadyRendered();
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

		this.updateIfAlreadyRendered();
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
		disconnectResizeObserver: function() {
			if(this.resizeObserver) {
				this.resizeObserver.disconnect();
				this.resizeObserver = null;
			}
		},

		// render again in case on_image_loaded was already called
		// this is called whenever we change this.detections to make sure
		// that the detections are drawn properly
		updateIfAlreadyRendered: function() {
			if(this.imageDims != null) {
				this.render();
			}
		},

		render: function() {
			let stage = new Konva.Stage({
				container: this.$refs.layer,
				width: this.imageDims.Width,
				height: this.imageDims.Height,
			});
			let layer = new Konva.Layer();

			// we want annotations to be stored in coordinates based on image natural width/height
			// but in the UI, image could be stretched to different width/height
			// so here we need to stretch the stage in the same way
			let getScale = () => {
				return Math.min(
					this.$refs.image.width / this.imageDims.Width,
					this.$refs.image.height / this.imageDims.Height,
				);
			};
			let rescaleLayer = () => {
				if(!this.$refs.layer || !this.$refs.image) {
					return;
				}
				let scale = getScale();
				stage.width(parseInt(scale*this.imageDims.Width));
				stage.height(parseInt(scale*this.imageDims.Height));
				layer.scaleX(scale);
				layer.scaleY(scale);
				layer.draw();
			};
			this.disconnectResizeObserver();
			this.resizeObserver = new ResizeObserver(rescaleLayer);
			this.resizeObserver.observe(this.$refs.image);
			rescaleLayer();
			let getPointerPosition = () => {
				let transform = layer.getAbsoluteTransform().copy();
				transform.invert();
				let pos = stage.getPointerPosition();
				return transform.point(pos);
			};

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
				let pos = getPointerPosition();
				// find closest unlabeled detection, using top-left of bounding box
				let bestDetection = null;
				let bestDistance = null;
				this.detections[this.frameIdx].forEach((detection) => {
					if(detection.TrackID) {
						return;
					}
					let dx = detection.Left - pos.x;
					let dy = detection.Top - pos.y;
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
	class="konva"
	>
</div>
		`,
	},
});
