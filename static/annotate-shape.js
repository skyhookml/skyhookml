import utils from './utils.js';
import AnnotateGenericUI from './annotate-generic-ui.js';

export default AnnotateGenericUI({
	data: function() {
		return {
			params: null,
			shapes: null,

			// index of currently selected shape, if any
			selectedIdx: null,

			keyupHandler: null,
		};
	},
	created: function() {
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
	},
	unmounted: function() {
		this.setKeyupHandler(null);
	},
	on_update: function() {
		this.shapes = [];
		for(let i = 0; i < this.numFrames; i++) {
			this.shapes.push([]);
		}
	},
	on_item_data: function(data) {
		if(data.length == 0) {
			return;
		}
		this.shapes = data.map((shapeList) => {
			return shapeList.map((shp) => this.decodeShape(shp));
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
		return this.shapes.map((shapeList) => {
			return shapeList.map((shape) => this.encodeShape(shape))
		});
	},
	methods: {
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
				width: this.imageDims.Width,
				height: this.imageDims.Height,
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
	template: {
		params: `
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
		im_below: `
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
		`,
	},
});
