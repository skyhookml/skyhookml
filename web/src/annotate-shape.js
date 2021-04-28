import utils from './utils.js';
import AnnotateGenericUI from './annotate-generic-ui.js';

export default AnnotateGenericUI({
	data: function() {
		return {
			params: null,
			shapes: null,

			// current category to use for labeling shapes
			category: '',

			// index of currently selected shape, if any
			selectedIdx: null,

			keyupHandler: null,
			resizeObserver: null,

			// handler functions set by render()
			cancelDrawHandler: null,
			deleteSelectionHandler: null,
		};
	},
	on_created_ready: function() {
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

		// call handlers on certain key presses
		this.setKeyupHandler((e) => {
			if(document.activeElement.tagName == 'INPUT') {
				return;
			}

			if(e.key === 'Escape' && this.cancelDrawHandler) {
				this.cancelDrawHandler();
			} else if(e.key === 'Delete' && this.deleteSelectionHandler) {
				this.deleteSelectionHandler();
			}
		});
	},
	destroyed: function() {
		this.setKeyupHandler(null);
		this.disconnectResizeObserver();
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
		let data = this.shapes.map((shapeList) => {
			return shapeList.map((shape) => this.encodeShape(shape))
		});
		let metadata = {
			CanvasDims: [this.imageDims.Width, this.imageDims.Height],
			Categories: this.params.Categories,
		};
		return [data, metadata];
	},
	methods: {
		disconnectResizeObserver: function() {
			if(this.resizeObserver) {
				this.resizeObserver.disconnect();
				this.resizeObserver = null;
			}
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
		updateCategories: function() {
			if(this.params.CategoriesStr == '') {
				this.params.Categories = [];
			} else {
				this.params.Categories = this.params.CategoriesStr.split(',');
			}
		},
		saveParams: function() {
			let request = {
				Params: JSON.stringify({
					Mode: this.params.Mode,
					Categories: this.params.Categories,
				}),
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
			let resizeLayer = null;
			let destroyResizeLayer = () => {
				if(resizeLayer) {
					resizeLayer.destroy();
					resizeLayer = null;
				}
			};

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
				if(resizeLayer) {
					resizeLayer.scaleX(scale);
					resizeLayer.scaleY(scale);
					resizeLayer.draw();
				}
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

			let konvaShapes = [];
			// curShape is set if we are currently drawing a new shape
			let curShape = null;

			let resetColors = () => {
				konvaShapes.forEach((kshp, idx) => {
					if(this.selectedIdx === idx) {
						kshp.stroke('orange');
					} else {
						kshp.stroke('red');
					}
				});
				layer.draw();
			};

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
						hitStrokeWidth: 20,
						fillEnabled: false,
						draggable: true,
					});
					kshp.myindex = idx;

					let updateShape = () => {
						shape.Points = [
							[parseInt(kshp.x()), parseInt(kshp.y())],
							[parseInt(kshp.x() + kshp.width()*kshp.scaleX()), parseInt(kshp.y() + kshp.height()*kshp.scaleY())],
						];
					};

					// called when a rectangle is selected
					// adds circles to the corners of the rectangle that the user can drag to resize the rectangle
					let handleResize = () => {
						destroyResizeLayer()
						resizeLayer = new Konva.Layer();
						resizeLayer.scaleX(getScale());
						resizeLayer.scaleY(getScale());
						stage.add(resizeLayer);

						// add circles at the four corners of the rectangle
						let offsets = [
							[0, 0],
							[1, 0],
							[0, 1],
							[1, 1],
						];
						let circles = [];
						let updateCircles = () => {
							circles.forEach((circle) => {
								circle.x(kshp.x()+circle.myoffset[0]*kshp.width());
								circle.y(kshp.y()+circle.myoffset[1]*kshp.height());
							});
						};
						offsets.forEach((offset) => {
							let circle = new Konva.Circle({
								x: kshp.x()+offset[0]*kshp.width(),
								y: kshp.y()+offset[1]*kshp.height(),
								radius: 10,
								fill: 'blue',
								stroke: 'black',
								strokeWidth: 2,
								draggable: true,
							});
							circle.myoffset = offset;
							circles.push(circle);

							circle.on('dragmove', (e) => {
								// If we move the right/bottom circle, then we just need to change the width/height.
								// But if we move the left/top circle, we need to update the x/y and correspondingly increase the width/height.
								if(offset[0] === 0) {
									kshp.width(kshp.width()+kshp.x()-circle.x());
									kshp.x(circle.x());
								} else {
									kshp.width(circle.x()-kshp.x());
								}
								if(offset[1] === 0) {
									kshp.height(kshp.height()+kshp.y()-circle.y());
									kshp.y(circle.y());
								} else {
									kshp.height(circle.y()-kshp.y());
								}
								updateCircles();
								updateShape();
								layer.draw();
								resizeLayer.draw();
							});

							resizeLayer.add(circle);
						});

						resizeLayer.draw();
					};

					kshp.on('click', (e) => {
						if(curShape) {
							return;
						}

						e.cancelBubble = true;
						handleResize();
						this.selectedIdx = kshp.myindex;
						resetColors();
					});

					kshp.on('dragstart', () => {
						// we don't want to worry about moving around the resize layer during drag operation
						// so instead we simply destroy it
						destroyResizeLayer();
					});
					kshp.on('dragend', () => {
						updateShape();
						// if this shape is selected, restore the resize layer
						if(this.selectedIdx === kshp.myindex) {
							handleResize();
						}
					});
				} else if(shape.Type == 'line') {
					kshp = new Konva.Line({
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
						if(curShape) {
							return;
						}

						e.cancelBubble = true;
						this.selectedIdx = kshp.myindex;
						resetColors();
					});

					kshp.on('dragend', updateShape);
				} else if(shape.Type == 'point') {
					kshp = new Konva.Circle({
						x: shape.Points[0][0],
						y: shape.Points[0][1],
						radius: 5,
						stroke: 'red',
						strokeWidth: 2,
						draggable: true,
					});

					let updateShape = () => {
						shape.Points = [[parseInt(kshp.x()), parseInt(kshp.y())]];
					};

					kshp.on('click', (e) => {
						if(curShape) {
							return;
						}

						e.cancelBubble = true;
						this.selectedIdx = kshp.myindex;
						resetColors();
					});

					kshp.on('dragend', updateShape);
				}

				kshp.on('mouseover', () => {
					if(curShape) {
						return;
					}
					if(this.selectedIdx === kshp.myindex) {
						// leave it under selected color instead of hover color
						return;
					}
					kshp.stroke('yellow');
					layer.draw();
				});
				kshp.on('mouseout', () => {
					resetColors();
				});

				layer.add(kshp);
				konvaShapes.push(kshp);
			};

			// add already existing shapes
			this.shapes[this.frameIdx].forEach((shape, idx) => {
				drawShape(shape, idx);
			});

			stage.add(layer);
			layer.draw();

			// mode-dependent handler in case user wants to cancel drawing a shape
			// (e.g., presses escape key)
			let cancelDrawHandler = null;

			if(this.params.Mode == 'box') {
				let updateRect = (x, y) => {
					let meta = curShape.meta;
					let width = Math.abs(meta.x - x);
					let height = Math.abs(meta.y - y);
					curShape.x(Math.min(meta.x, x));
					curShape.y(Math.min(meta.y, y));
					curShape.width(width);
					curShape.height(height);
				};
				stage.on('click', () => {
					if(resizeLayer) {
						destroyResizeLayer()
						this.selectedIdx = null;
						resetColors();
						return;
					}

					var pos = getPointerPosition();
					if(curShape == null) {
						curShape = new Konva.Rect({
							x: pos.x,
							y: pos.y,
							width: 1,
							height: 1,
							stroke: 'yellow',
							strokeWidth: 3,
						});
						curShape.meta = {x: pos.x, y: pos.y};
						layer.add(curShape);
						layer.draw();
					} else {
						updateRect(pos.x, pos.y);

						let shape = {
							Type: 'box',
							Points: [
								[parseInt(curShape.x()), parseInt(curShape.y())],
								[parseInt(curShape.x()+curShape.width()), parseInt(curShape.y()+curShape.height())],
							],
							Category: this.category,
							TrackID: '',
						};
						this.shapes[this.frameIdx].push(shape);
						drawShape(shape, this.shapes[this.frameIdx].length-1);

						curShape.destroy();
						curShape = null;
						layer.draw();
					}
				});
				stage.on('mousemove', () => {
					if(curShape == null) {
						return;
					}
					var pos = getPointerPosition();
					updateRect(pos.x, pos.y);
					layer.batchDraw();
				});

				this.cancelDrawHandler = () => {
					if(curShape === null) {
						return;
					}
					curShape.destroy();
					curShape = null;
					layer.draw();
				};
			} else if(this.params.Mode == 'line') {
				let updateLine = (x, y) => {
					let pts = curShape.points();
					curShape.points([pts[0], pts[1], x, y]);
				};
				stage.on('click', () => {
					if(resizeLayer) {
						destroyResizeLayer()
						this.selectedIdx = null;
						layer.draw();
						return;
					}

					var pos = getPointerPosition();
					if(curShape == null) {
						curShape = new Konva.Line({
							points: [pos.x, pos.y, pos.x+1, pos.y+1],
							stroke: 'yellow',
							strokeWidth: 3,
						});
						layer.add(curShape);
						layer.draw();
					} else {
						updateLine(pos.x, pos.y);

						let pts = curShape.points();
						let shape = {
							Type: 'line',
							Points: [
								[parseInt(pts[0]), parseInt(pts[1])],
								[parseInt(pts[2]), parseInt(pts[3])],
							],
							Category: this.category,
							TrackID: '',
						};
						this.shapes[this.frameIdx].push(shape);
						drawShape(shape, this.shapes[this.frameIdx].length-1);

						curShape.destroy();
						curShape = null;
						layer.draw();
					}
				});
				stage.on('mousemove', () => {
					if(curShape == null) {
						return;
					}
					var pos = getPointerPosition();
					updateLine(pos.x, pos.y);
					layer.batchDraw();
				});

				this.cancelDrawHandler = () => {
					if(curShape === null) {
						return;
					}
					curShape.destroy();
					curShape = null;
					layer.draw();
				}
			} else if(this.params.Mode == 'point') {
				stage.on('click', () => {
					if(resizeLayer) {
						destroyResizeLayer()
						this.selectedIdx = null;
						layer.draw();
						return;
					}

					var pos = getPointerPosition();
					let shape = {
						Type: 'point',
						Points: [[parseInt(pos.x), parseInt(pos.y)]],
						Category: this.category,
						TrackID: '',
					};
					this.shapes[this.frameIdx].push(shape);
					drawShape(shape, this.shapes[this.frameIdx].length-1);
					layer.draw();
				});

				this.cancelDrawHandler = () => {
					if(curShape === null) {
						return;
					}
					curShape.destroy();
					curShape = null;
					layer.draw();
				}
			}

			// initialize a handler for deleting selected shapes
			this.deleteSelectionHandler = () => {
				if(this.selectedIdx === null) {
					return;
				}
				this.shapes[this.frameIdx].splice(this.selectedIdx, 1);
				let kshp = konvaShapes[this.selectedIdx];
				konvaShapes.splice(this.selectedIdx, 1);
				konvaShapes.forEach((kshp, idx) => {
					kshp.myindex = idx;
				});
				kshp.destroy();
				destroyResizeLayer();
				this.selectedIdx = null;
				layer.draw();
			};
		},
	},
	template: {
		params: `
<form class="row g-1 align-items-center" v-on:submit.prevent="saveParams">
	<div class="col-auto">
		<label>Mode</label>
	</div>
	<div class="col-auto">
		<select class="form-select" v-model="params.Mode" @change="render">
			<option value="box">Box</option>
			<option value="point">Point</option>
			<option value="line">Line</option>
			<option value="polygon">Polygon</option>
		</select>
	</div>
	<div class="col-auto">
		<template v-if="params.Mode == 'box'">
			<i class="bi bi-question-circle" data-bs-toggle="tooltip" title="Box mode: click twice to draw a box. Escape to cancel current drawing, click to select box, Delete to delete selection."></i>
		</template>
	</div>
	<div class="col-auto">
		<label>Categories (comma-separated)</label>
	</div>
	<div class="col-auto">
		<input class="form-control" type="text" v-model="params.CategoriesStr" @change="updateCategories">
	</div>
	<div class="col-auto">
		<button type="submit" class="btn btn-primary">Save Settings</button>
	</div>
</form>
		`,
		im_above: `
<div class="row g-1 align-items-center">
	<div class="col-auto">
		<label>Category:</label>
	</div>
	<div class="col-auto">
		<select class="form-select form-select-sm" v-model="category">
			<option value="">None</option>
			<template v-for="category in params.Categories">
				<option :key="category" :value="category">{{ category }}</option>
			</template>
		</select>
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
		im_below: `
<div v-if="selectedIdx != null && selectedIdx >= 0 && selectedIdx < shapes[frameIdx].length" class="mb-2">
	<p>
		<strong>Selection: {{ shapes[frameIdx][selectedIdx].Type }} ({{ shapes[frameIdx][selectedIdx].Points }})</strong>
		<button v-if="deleteSelectionHandler" type="button" class="btn btn-sm btn-danger" v-on:click="deleteSelectionHandler">Delete</button>
	</p>
	<div class="small-container">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Category</label>
			<div class="col-sm-10">
				<select class="form-select" v-model="shapes[frameIdx][selectedIdx].Category">
					<option value="">None</option>
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
