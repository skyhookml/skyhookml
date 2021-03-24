import utils from './utils.js';

export default function(impl) {
	let component = {
		data: function() {
			let data = {
				// annoset.DataType can be shape, but can also be detection
				// the source data type can be image or video
				annoset: null,
				dataType: null,
				source: null,
				sourceType: null,
				url: '',

				// dimensions of currently loaded image
				imageDims: null,

				// the item metadata and annotation response for the current source item
				itemMeta: null,
				response: null,

				// current frame index that we're looking at (always 0 for image source)
				frameIdx: null,
				numFrames: 0,

				// list of keys for iteration over previously labeled items
				keyList: null,
				itemIdx: 0,
			};
			if(impl.data) {
				let impl_data = impl.data.call(this);
				for(var key in impl_data) {
					data[key] = impl_data[key];
				}
			}
			return data;
		},
		created: function() {
			if(impl.created) {
				impl.created.call(this);
			}
			const setID = this.$route.params.setid;
			utils.request(this, 'GET', '/annotate-datasets/'+setID, null, (annoset) => {
				this.annoset = annoset;
				this.dataType = annoset.Dataset.DataType;
				this.source = annoset.Inputs[0];
				this.sourceType = this.source.DataType;
				this.url = '/annotate-datasets/'+this.annoset.ID+'/annotate';
				if(impl.on_created_ready) {
					impl.on_created_ready.call(this);
				}
				this.update();
			});
		},
		mounted: function() {
			if(impl.mounted) {
				impl.mounted.call(this);
			}
		},
		destroyed: function() {
			if(impl.destroyed) {
				impl.destroyed.call(this);
			}
		},
		methods: {
			resetFrame: function() {
				this.imageDims = null;
				this.frameIdx = null;
				this.selectedIdx = null;

				if(impl.resetFrame) {
					impl.resetFrame.call(this);
				}
			},
			resetItem: function() {
				this.resetFrame();
				this.itemMeta = null;
				this.response = null;
				this.numFrames = 0;

				if(impl.resetItem) {
					impl.resetItem.call(this);
				}
			},
			update: function() {
				let url = this.url;
				if(this.keyList != null) {
					url += '?key='+this.keyList[this.itemIdx];
				}
				let response, itemMeta;
				utils.request(this, 'GET', url, null, (data) => {
					response = data;
				}).then(() => {
					return utils.request(this, 'GET', '/datasets/'+this.source.ID+'/items/'+response.Key+'/get?format=meta', null, (data) => {
						itemMeta = data;
					});
				}).then(() => {
					this.resetItem();
					this.itemMeta = itemMeta;

					if(this.sourceType == 'image') {
						this.numFrames = 1;
					} else if(this.sourceType == 'video') {
						this.numFrames = parseInt(this.itemMeta.Duration * this.itemMeta.Framerate[0] / this.itemMeta.Framerate[1]);
					}

					Vue.nextTick(() => {
						this.response = response;
						this.frameIdx = 0;

						if(impl.on_update) {
							impl.on_update.call(this);
						}

						if(this.response.IsExisting) {
							let params = {
								format: 'json',
								t: new Date().getTime(),
							};
							utils.request(this, 'GET', '/datasets/'+this.annoset.Dataset.ID+'/items/'+this.response.Key+'/get', params, (data) => {
								if(impl.on_item_data) {
									impl.on_item_data.call(this, data, this.itemMeta);
								}
							});
						}
					});
				});
			},
			imageLoaded: function() {
				this.imageDims = {
					Width: parseInt(this.$refs.image.naturalWidth),
					Height: parseInt(this.$refs.image.naturalHeight),
				};
				if(impl.on_image_loaded) {
					impl.on_image_loaded.call(this);
				}
			},
			getNewItem: function() {
				this.keyList = null;
				this.itemIdx = 0;
				this.update();
			},
			getOldItem: function(i) {
				if(!this.keyList) {
					utils.request(this, 'GET', '/datasets/'+this.annoset.Dataset.ID+'/items', null, (items) => {
						if(!items || items.length == 0) {
							return;
						}
						this.keyList = items.map((item) => item.Key);
						this.getOldItem(0);
					});
					return;
				}

				this.itemIdx = (i + this.keyList.length) % this.keyList.length;
				this.update();
			},
			getFrame: function(i) {
				this.resetFrame();
				// wait until next tick so that the <img> will be deleted
				// this ensures the onload will correctly call imageLoaded to populate imageDims
				Vue.nextTick(() => {
					this.frameIdx = (i + this.numFrames) % this.numFrames;
				});
			},
			// advance to the next frame, or annotate and proceed to next item if at the end
			finishFrame: function() {
				let frameIdx = this.frameIdx + 1;
				if(frameIdx >= this.numFrames) {
					this.annotateItem();
					return;
				}
				this.getFrame(frameIdx);
			},
			annotateItem: function() {
				let data = impl.getAnnotateData.call(this);
				let request = {
					Key: this.response.Key,
					Data: JSON.stringify(data[0]),
					Format: 'json',
					Metadata: JSON.stringify(data[1]),
				};
				utils.request(this, 'POST', this.url, JSON.stringify(request), () => {
					if(this.keyList == null) {
						this.getNewItem();
					} else {
						this.getOldItem(this.itemIdx+1);
					}
				});
			},
		},
	};
	let template = `
	<div class="flex-container el-high">
		<template v-if="annoset != null">
			<div class="mb-2">
				[PARAMS]
			</div>

			<div class="row align-items-center g-1 mb-2">
				<div class="col-auto">
					<button v-on:click="getOldItem(itemIdx-1)" type="button" class="btn btn-primary">Prev</button>
				</div>
				<div class="col-auto">
					<template v-if="response != null">
						<span>{{ response.Key }}</span>
						<span v-if="keyList != null">({{ itemIdx }} of {{ keyList.length }})</span>
					</template>
				</div>
				<div class="col-auto">
					<button v-on:click="getOldItem(itemIdx+1)" type="button" class="btn btn-primary">Next</button>
				</div>
				<div class="col-auto">
					<button v-on:click="getNewItem" type="button" class="btn btn-primary">New</button>
				</div>
				<div class="col-auto" v-if="response != null">
					<button type="button" class="btn btn-primary" v-on:click="annotateItem">Done</button>
				</div>
			</div>

			[IM_ABOVE]

			<div class="my-2 flex-content canvas-container">
				<template v-if="frameIdx != null">
					<template v-if="sourceType == 'video'">
						<img :src="'/datasets/'+annoset.Inputs[0].ID+'/items/'+response.Key+'/get-video-frame?idx='+frameIdx" class="fill-img" @load="imageLoaded" ref="image" />
					</template>
					<template v-else>
						<img :src="'/datasets/'+annoset.Inputs[0].ID+'/items/'+response.Key+'/get?format=jpeg'" class="fill-img" @load="imageLoaded" ref="image" />
					</template>
				</template>

				[IM_AFTER]
			</div>

			[IM_BELOW]

			<div v-if="sourceType == 'video'" class="row align-items-center g-1">
				<div class="col-auto">
					<button v-on:click="getFrame(frameIdx-1)" type="button" class="btn btn-primary">Prev Frame</button>
				</div>
				<div class="col-auto">
					<template v-if="response != null">
						Frame {{ frameIdx }} / {{ numFrames }}
					</template>
				</div>
				<div class="col-auto">
					<button v-on:click="getFrame(frameIdx+1)" type="button" class="btn btn-primary">Next Frame</button>
				</div>
			</div>
		</template>
	</div>`;
	for(var key in impl.methods) {
		component.methods[key] = impl.methods[key];
	}

	template = template.replace('[PARAMS]', impl.template.params ? impl.template.params : '');
	template = template.replace('[IM_ABOVE]', impl.template.im_above ? impl.template.im_above : '');
	template = template.replace('[IM_AFTER]', impl.template.im_after ? impl.template.im_after : '');
	template = template.replace('[IM_BELOW]', impl.template.im_below ? impl.template.im_below : '');
	component.template = template;
	return component;
};
