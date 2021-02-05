import utils from './utils.js';

export default {
	data: function() {
		return {
			// annoset.DataType can be shape, but can also be detection
			// the source data type can be image or video
			annoset: null,
			dataType: null,
			source: null,
			sourceType: null,
			url: '',

			// config for this annotation tool
			params: null,

			// dimensions of currently loaded image
			imageMeta: null,

			// the item metadata and annotation response for the current source item
			itemMeta: null,
			response: null,

			// current frame index that we're looking at (always 0 for image source)
			frameIdx: null,
			numFrames: 0,

			// shapes for current image sequence
			shapes: null,

			// index of currently selected shape, if any
			selectedIdx: null,

			// list of keys for iteration over previously labeled items
			keyList: null,
			itemIdx: 0,

			keyupHandler: null,
		};
	},
	created: function() {
		const setID = this.$route.params.setid;
		utils.request(this, 'GET', '/annotate-datasets/'+setID, null, (annoset) => {
			this.annoset = annoset;
			this.dataType = annoset.Dataset.DataType;
			this.source = annoset.Inputs[0];
			this.sourceType = this.source.DataType;
			this.url = '/annotate-datasets/'+this.annoset.ID+'/annotate';
			let params;
			try {
				params = JSON.parse(this.annoset.Params);
			} catch(e) {}
			if(!params) {
				params = {};
			}
			if(!params.Mode) {
				params.Mode = 'box';
			}
			if(!params.Categories) {
				params.Categories = [];
				params.CategoriesStr = '';
			} else {
				params.CategoriesStr = params.Categories.join(',');
			}
			this.params = params;
			this.update();
		});
	},
	unmounted: function() {
		this.setKeyupHandler(null);
	},
	methods: {
		resetFrame: function() {
			this.imageMeta = null;
			this.frameIdx = null;
			this.selectedIdx = null;
		},
		resetItem: function() {
			this.resetFrame();
			this.itemMeta = null;
			this.response = null;
			this.numFrames = 0;
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
				this.shapes = [];

				// initialize shapes for each frame
				if(this.sourceType == 'image') {
					this.numFrames = 1;
				} else if(this.sourceType == 'video') {
					this.numFrames = parseInt(this.itemMeta.Duration * this.itemMeta.Framerate[0] / this.itemMeta.Framerate[1]);
				}
				for(let i = 0; i < this.numFrames; i++) {
					this.shapes.push([]);
				}

				Vue.nextTick(() => {
					this.response = response;
					this.frameIdx = 0;

					if(this.response.IsExisting) {
						let params = {
							format: 'json',
							t: new Date().getTime(),
						};
						utils.request(this, 'GET', '/datasets/'+this.annoset.Dataset.ID+'/items/'+this.response.Key+'/get', params, (data) => {
							if(data.length == 0) {
								return;
							}
							this.shapes = data.map((shapeList) => {
								return shapeList.map((shp) => this.decodeShape(shp));
							});

							// update if we already rendered before setting shapes
							if(this.imageMeta != null) {
								this.imageLoaded();
							}
						});
					}
				});
			});
		},
		decodeShape: function(shape) {
			let shp = {};
			if(this.dataType === 'shape') {
				shp.Type = shape.Type;
				shp.Points = shape.Points;
			} else if(this.dataType === 'detection') {
				shp.Type = 'box';
				shp.Points = [[shape.Left, shape.Top], [shape.Right, shape.Bottom]];
			}
			shp.Category = (shape.Category) ? shape.Category : '';
			shp.TrackID = (shape.TrackID) ? shape.TrackID : '';
			return shp;
		},
		encodeShape: function(shape) {
			let shp = {};
			if(this.dataType === 'shape') {
				shp.Type = shape.Type;
				shp.Points = shape.Points;
			} else if(this.dataType === 'detection') {
				shp.Left = shape.Points[0][0];
				shp.Top = shape.Points[0][1];
				shp.Right = shape.Points[1][0];
				shp.Bottom = shape.Points[1][1];
			}
			if(shape.Category !== '') {
				shp.Category = shape.Category;
			}
			if(shape.TrackID !== '') {
				shp.TrackID = parseInt(shape.TrackID);
			}
			return shp;
		},
		imageLoaded: function() {
			this.imageMeta = {
				Width: parseInt(this.$refs.image.width),
				Height: parseInt(this.$refs.image.height),
			};
			Vue.nextTick(() => {
				this.render();
			});
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
			// this ensures the onload will correctly call imageLoaded to populate imageMeta
			Vue.nextTick(() => {
				this.frameIdx = (i + this.numFrames) % this.numFrames;
			});
		},
		annotateItem: function() {
			let shapes = this.shapes.map((shapeList) => {
				return shapeList.map((shape) => this.encodeShape(shape))
			});
			let request = {
				Key: this.response.Key,
				Data: JSON.stringify(shapes),
				Format: 'json',
				Metadata: JSON.stringify({
					CanvasDims: [this.imageMeta.Width, this.imageMeta.Height],
				}),
			};
			utils.request(this, 'POST', this.url, JSON.stringify(request), () => {
				if(this.keyList == null) {
					this.getNewItem();
				} else {
					this.getOldItem(this.itemIdx+1);
				}
			});
		},
		updateCategories: function() {
			if(this.params.CategoriesStr == '') {
				this.params.Categories = [];
			} else {
				this.params.Categories = this.params.CategoriesStr.split(',');
			}
		},
		saveParams: function() {
			let request = {
				Params: JSON.stringify(this.params),
			};
			utils.request(this, 'POST', '/annotate-datasets/'+this.annoset.ID, JSON.stringify(request));
		},
		setKeyupHandler: function(handler) {
			if(this.keyupHandler != null) {
				this.$parent.$off('keyup', this.keyupHandler);
				this.keyupHandler = null;
			}
			if(handler != null) {
				this.keyupHandler = handler;
				this.$parent.$on('keyup', this.keyupHandler);
			}
		},
		render: function() {
			let stage = new Konva.Stage({
				container: this.$refs.layer,
				width: this.imageMeta.Width,
				height: this.imageMeta.Height,
			});
			let layer = new Konva.Layer();

			let tr = new Konva.Transformer({
				nodes: [],
				rotateEnabled: false,
			});
			layer.add(tr);

			let konvaShapes = [];

			let drawShape = (shape, idx) => {
				let kshp = null;

				if(shape.Type == 'box') {
					kshp = new Konva.Rect({
						x: shape.Points[0][0],
						y: shape.Points[0][1],
						width: shape.Points[1][0]-shape.Points[0][0],
						height: shape.Points[1][1]-shape.Points[0][1],
						stroke: 'red',
						strokeWidth: 3,
						draggable: true,
					});

					let updateShape = () => {
						shape.Points = [
							[parseInt(kshp.x()), parseInt(kshp.y())],
							[parseInt(kshp.x() + kshp.width()*kshp.scaleX()), parseInt(kshp.y() + kshp.height()*kshp.scaleY())],
						];
					};

					kshp.on('click', (e) => {
						e.cancelBubble = true;

						// select this rect in the transformer
						tr.nodes([kshp]);
						layer.draw();

						this.selectedIdx = idx;
					});

					kshp.on('transformend', updateShape);
					kshp.on('dragend', updateShape);
				} else if(shape.Type == 'line') {
					let kshp = new Konva.Line({
						points: [shape.Points[0][0], shape.Points[0][1], shape.Points[1][0], shape.Points[1][1]],
						stroke: 'red',
						strokeWidth: 3,
						draggable: true,
					});

					let updateShape = () => {
						let pts = kshp.points();
						shape.Points = [
							[parseInt(kshp.x()+pts[0]), parseInt(kshp.y()+pts[1])],
							[parseInt(kshp.x()+pts[2]), parseInt(kshp.y()+pts[3])],
						];
					};

					kshp.on('click', (e) => {
						e.cancelBubble = true;
						this.selectedIdx = idx;
					});

					kshp.on('dragend', updateShape);
				}

				layer.add(kshp);
				konvaShapes.push(kshp);
			};

			// add already existing shapes
			this.shapes[this.frameIdx].forEach((shape, idx) => {
				drawShape(shape, idx);
			});

			stage.add(layer);
			layer.draw();

			if(this.params.Mode == 'box') {
				let curRect = null;
				let updateRect = (x, y) => {
					let meta = curRect.meta;
					let width = Math.abs(meta.x - x);
					let height = Math.abs(meta.y - y);
					curRect.x(Math.min(meta.x, x));
					curRect.y(Math.min(meta.y, y));
					curRect.width(width);
					curRect.height(height);
				};
				stage.on('click', () => {
					if(tr.nodes().length > 0) {
						tr.nodes([]);
						this.selectedIdx = null;
						layer.draw();
						return;
					}

					var pos = stage.getPointerPosition();
					if(curRect == null) {
						curRect = new Konva.Rect({
							x: pos.x,
							y: pos.y,
							width: 1,
							height: 1,
							stroke: 'yellow',
							strokeWidth: 3,
						});
						curRect.meta = {x: pos.x, y: pos.y};
						layer.add(curRect);
						layer.draw();
					} else {
						updateRect(pos.x, pos.y);

						let shape = {
							Type: 'box',
							Points: [
								[parseInt(curRect.x()), parseInt(curRect.y())],
								[parseInt(curRect.x()+curRect.width()), parseInt(curRect.y()+curRect.height())],
							],
							Category: '',
							TrackID: '',
						};
						this.shapes[this.frameIdx].push(shape);
						drawShape(shape, this.shapes[this.frameIdx].length-1);

						curRect.destroy();
						curRect = null;
						layer.draw();
					}
				});
				stage.on('mousemove', () => {
					if(curRect == null) {
						return;
					}
					var pos = stage.getPointerPosition();
					updateRect(pos.x, pos.y);
					layer.batchDraw();
				});

				this.setKeyupHandler((e) => {
					if(document.activeElement.tagName == 'INPUT') {
						return;
					}

					if(e.key === 'Escape') {
						if(curRect === null) {
							return;
						}
						curRect.destroy();
						curRect = null;
						layer.draw();
					} else if(e.key === 'Delete') {
						if(this.selectedIdx === null) {
							return;
						}
						this.shapes[this.frameIdx].splice(this.selectedIdx, 1);
						let kshp = konvaShapes[this.selectedIdx];
						konvaShapes.splice(this.selectedIdx, 1);
						kshp.destroy();
						tr.nodes([]);
						this.selectedIdx = null;
						layer.draw();
					}
				});
			} else if(this.params.Mode == 'line') {
				let curLine = null;
				let updateLine = (x, y) => {
					let pts = curLine.points();
					curLine.points([pts[0], pts[1], x, y]);
				};
				stage.on('click', () => {
					if(tr.nodes().length > 0) {
						tr.nodes([]);
						this.selectedIdx = null;
						layer.draw();
						return;
					}

					var pos = stage.getPointerPosition();
					if(curLine == null) {
						curLine = new Konva.Line({
							points: [pos.x, pos.y, pos.x+1, pos.y+1],
							stroke: 'yellow',
							strokeWidth: 3,
						});
						layer.add(curLine);
						layer.draw();
					} else {
						updateLine(pos.x, pos.y);

						let pts = curLine.points();
						let shape = {
							Type: 'line',
							Points: [
								[parseInt(pts[0]), parseInt(pts[1])],
								[parseInt(pts[2]), parseInt(pts[3])],
							],
							Category: '',
							TrackID: '',
						};
						this.shapes[this.frameIdx].push(shape);
						drawShape(shape, this.shapes[this.frameIdx].length-1);

						curLine.destroy();
						curLine = null;
						layer.draw();
					}
				});
				stage.on('mousemove', () => {
					if(curLine == null) {
						return;
					}
					var pos = stage.getPointerPosition();
					updateLine(pos.x, pos.y);
					layer.batchDraw();
				});

				this.setKeyupHandler((e) => {
					if(document.activeElement.tagName == 'INPUT') {
						return;
					}

					if(e.key === 'Escape') {
						if(curLine === null) {
							return;
						}
						curLine.destroy();
						curLine = null;
						layer.draw();
					} else if(e.key === 'Delete') {
						if(this.selectedIdx === null) {
							return;
						}
						this.shapes[this.frameIdx].splice(this.selectedIdx, 1);
						let kshp = konvaShapes[this.selectedIdx];
						konvaShapes.splice(this.selectedIdx, 1);
						kshp.destroy();
						tr.nodes([]);
						this.selectedIdx = null;
						layer.draw();
					}
				});
			}
		},
	},
	template: `
<div>
	<template v-if="annoset != null">
		<div>
			<form class="form-inline" v-on:submit.prevent="saveParams">
				<label class="my-1 mx-1">Mode</label>
				<select class="form-control my-1 mx-1" v-model="params.Mode" @change="render">
					<option value="box">Box</option>
					<option value="point">Point</option>
					<option value="line">Line</option>
					<option value="polygon">Polygon</option>
				</select>

				<label class="my-1 mx-1">Categories (comma-separated)</label>
				<input class="form-control my-1 mx-1" type="text" v-model="params.CategoriesStr" @change="updateCategories">

				<button type="submit" class="btn btn-primary my-1 mx-1">Save Settings</button>
			</form>
		</div>

		<div class="form-row align-items-center">
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

		<div class="canvas-container">
			<template v-if="frameIdx != null">
				<template v-if="sourceType == 'video'">
					<img :src="'/datasets/'+annoset.Inputs[0].ID+'/items/'+response.Key+'/get-video-frame?idx='+frameIdx" @load="imageLoaded" ref="image" />
				</template>
				<template v-else>
					<img :src="'/datasets/'+annoset.Inputs[0].ID+'/items/'+response.Key+'/get?format=jpeg'" @load="imageLoaded" ref="image" />
				</template>
			</template>
			<div
				v-if="imageMeta != null"
				class="conva"
				ref="layer"
				:style="{
					width: imageMeta.Width+'px',
					height: imageMeta.Height+'px',
				}"
				>
			</div>
		</div>

		<div v-if="selectedIdx != null && selectedIdx >= 0 && selectedIdx < shapes[frameIdx].length">
			<p><strong>Selection: {{ shapes[frameIdx][selectedIdx].Type }} ({{ shapes[frameIdx][selectedIdx].Points }})</strong></p>
			<div class="small-container">
				<div class="form-group row">
					<label class="col-sm-2 col-form-label">Category</label>
					<div class="col-sm-10">
						<select class="form-control" v-model="shapes[frameIdx][selectedIdx].Category">
							<option :key="''" value="">None</option>
							<template v-for="category in params.Categories">
								<option :key="category" :value="category">{{ category }}</option>
							</template>
						</select>
					</div>
				</div>
				<div class="form-group row">
					<label class="col-sm-2 col-form-label">Track ID</label>
					<div class="col-sm-10">
						<input type="text" class="form-control" v-model="shapes[frameIdx][selectedIdx].TrackID" />
					</div>
				</div>
			</div>
		</div>

		<div v-if="sourceType == 'video'" class="form-row align-items-center">
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
</div>
	`,
};
