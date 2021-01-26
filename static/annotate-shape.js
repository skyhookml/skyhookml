import utils from './utils.js';

export default {
	data: function() {
		return {
			annoset: null,
			params: null,
			url: '',

			response: null,
			imageMeta: null,
			shapes: [],

			// index of currently selected shape, if any
			selectedIdx: null,

			// list of keys for iteration over previously labeled items
			keyList: null,
			curIndex: 0,

			keyupHandler: null,
		};
	},
	created: function() {
		const setID = this.$route.params.setid;
		utils.request(this, 'GET', '/annotate-datasets/'+setID, null, (annoset) => {
			this.annoset = annoset;
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
			utils.request(this, 'GET', this.url, null, this.update);
		});
	},
	unmounted: function() {
		this.setKeyupHandler(null);
	},
	methods: {
		update: function(response) {
			this.response = null;
			this.imageMeta = null;

			Vue.nextTick(() => {
				this.response = response;
				this.shapes = [];
				this.selectedIdx = null;

				if(this.response.IsExisting) {
					let params = {
						format: 'json',
						t: new Date().getTime(),
					};
					utils.request(this, 'GET', '/datasets/'+this.annoset.Dataset.ID+'/items/'+this.response.Key+'/get', params, (data) => {
						if(data.length == 0) {
							return;
						}
						let shapes = data[0];
						shapes.forEach((shp) => this.addShapeMetadataAttributes(shp));
						this.shapes = shapes;

						// update if we already rendered before setting shapes
						if(this.imageMeta != null) {
							this.imageLoaded();
						}
					});
				}
			});
		},
		addShapeMetadataAttributes: function(shape) {
			if(!shape.hasOwnProperty('Category')) {
				shape.Category = '';
			}
			if(!shape.hasOwnProperty('TrackID')) {
				shape.TrackID = '';
			}
		},
		encodableShape: function(shape) {
			let shp = {
				'Type': shape.Type,
				'Points': shape.Points,
			};
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
		getNew: function() {
			this.keyList = null;
			this.curIndex = 0;
			utils.request(this, 'GET', this.url, null, this.update);
		},
		getOld: function(i) {
			if(!this.keyList) {
				utils.request(this, 'GET', '/datasets/'+this.annoset.Dataset.ID+'/items', null, (items) => {
					if(!items || items.length == 0) {
						return;
					}
					this.keyList = items.map((item) => item.Key);
					this.getOld(0);
				});
				return;
			}

			this.curIndex = (i + this.keyList.length) % this.keyList.length;
			utils.request(this, 'GET', this.url+'?key='+this.keyList[this.curIndex], null, this.update);
		},
		annotate: function() {
			let shapes = this.shapes.map((shape) => this.encodableShape(shape));
			let request = {
				Key: this.response.Key,
				Data: JSON.stringify([shapes]),
				Format: 'json',
				Metadata: JSON.stringify({
					CanvasDims: [this.imageMeta.Width, this.imageMeta.Height],
				}),
			};
			utils.request(this, 'POST', this.url, JSON.stringify(request), () => {
				if(this.keyList == null) {
					this.getNew();
				} else {
					this.getOld(this.curIndex+1);
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
			this.shapes.forEach((shape, idx) => {
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
						};
						this.addShapeMetadataAttributes(shape);
						this.shapes.push(shape);
						drawShape(shape, this.shapes.length-1);

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
						this.shapes.splice(this.selectedIdx, 1);
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
						};
						this.addShapeMetadataAttributes(shape);
						this.shapes.push(shape);
						drawShape(shape, this.shapes.length-1);

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
						this.shapes.splice(this.selectedIdx, 1);
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
		<div class="canvas-container">
			<img v-if="response != null" :src="'/datasets/'+annoset.Inputs[0].ID+'/items/'+response.Key+'/get?format=jpeg'" @load="imageLoaded" ref="image" />
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

		<div v-if="selectedIdx != null && selectedIdx >= 0 && selectedIdx < shapes.length">
			<p><strong>Selection: {{ shapes[selectedIdx].Type }} ({{ shapes[selectedIdx].Points }})</strong></p>
			<div class="small-container">
				<div class="form-group row">
					<label class="col-sm-2 col-form-label">Category</label>
					<div class="col-sm-10">
						<select class="form-control" v-model="shapes[selectedIdx].Category">
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
						<input type="text" class="form-control" v-model="shapes[selectedIdx].TrackID" />
					</div>
				</div>
			</div>
		</div>

		<div class="form-row align-items-center">
			<div class="col-auto">
				<button v-on:click="getOld(curIndex-1)" type="button" class="btn btn-primary">Prev</button>
			</div>
			<div class="col-auto">
				<template v-if="response != null">
					<span>{{ response.key }}</span>
					<span v-if="keyList != null">({{ curIndex }} of {{ keyList.length }})</span>
				</template>
			</div>
			<div class="col-auto">
				<button v-on:click="getOld(curIndex+1)" type="button" class="btn btn-primary">Next</button>
			</div>
			<div class="col-auto">
				<button v-on:click="getNew" type="button" class="btn btn-primary">New</button>
			</div>
			<div class="col-auto" v-if="response != null">
				<button type="button" class="btn btn-primary" v-on:click="annotate">Done</button>
			</div>
		</div>
	</template>
</div>
	`,
};
