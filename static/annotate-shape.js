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

			// list of keys for iteration over previously labeled items
			keyList: null,
			curIndex: 0,
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
			this.params = params;
			utils.request(this, 'GET', this.url, null, this.update);
		});
	},
	methods: {
		update: function(response) {
			this.response = null;
			this.imageMeta = null;

			Vue.nextTick(() => {
				this.response = response;
				this.shapes = [];

				if(this.response.IsExisting) {
					let params = {
						format: 'json',
						t: new Date().getTime(),
					};
					utils.request(this, 'GET', '/datasets/'+this.annoset.Dataset.ID+'/items/'+this.response.Key+'/get', params, (data) => {
						if(data.length == 0) {
							return;
						}
						this.shapes = data[0];

						// update if we already rendered before setting shapes
						if(this.imageMeta != null) {
							this.imageLoaded();
						}
					});
				}
			});
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
			var request = {
				Key: this.response.Key,
				Data: JSON.stringify([this.shapes]),
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
		saveParams: function() {
			utils.request(this, 'POST', '/annotate-datasets/'+this.annoset.ID, {Params: JSON.stringify(this.params)});
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
				rotationEnabled: false,
			});
			layer.add(tr);

			let drawShape = (shape) => {
				if(shape.Type == 'box') {
					let rect = new Konva.Rect({
						x: shape.Points[0][0],
						y: shape.Points[0][1],
						width: shape.Points[1][0]-shape.Points[0][0],
						height: shape.Points[1][1]-shape.Points[0][1],
						stroke: 'red',
						strokeWidth: 3,
						draggable: true,
					});
					layer.add(rect);

					let updateShape = () => {
						shape.Points = [
							[parseInt(rect.x()), parseInt(rect.y())],
							[parseInt(rect.x() + rect.width()*rect.scaleX()), parseInt(rect.y() + rect.height()*rect.scaleY())],
						];
					};

					rect.on('click', (e) => {
						e.cancelBubble = true;

						// select this rect in the transformer
						tr.nodes([rect]);
						layer.draw();
					});

					rect.on('transformend', updateShape);
					rect.on('dragend', updateShape);
				} else if(shape.Type == 'line') {
					let line = new Konva.Line({
						points: [shape.Points[0][0], shape.Points[0][1], shape.Points[1][0], shape.Points[1][1]],
						stroke: 'red',
						strokeWidth: 3,
						draggable: true,
					});
					layer.add(line);

					let updateShape = () => {
						let pts = line.points();
						shape.Points = [
							[parseInt(line.x()+pts[0]), parseInt(line.y()+pts[1])],
							[parseInt(line.x()+pts[2]), parseInt(line.y()+pts[3])],
						];
					};

					line.on('dragend', updateShape);
				}
			};

			// add already existing shapes
			this.shapes.forEach((shape) => {
				drawShape(shape);
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
					tr.nodes([]);

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
						this.shapes.push(shape);
						drawShape(shape);

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
			} else if(this.params.Mode == 'line') {
				let curLine = null;
				let updateLine = (x, y) => {
					let pts = curLine.points();
					curLine.points([pts[0], pts[1], x, y]);
				};
				stage.on('click', () => {
					tr.nodes([]);

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
						this.shapes.push(shape);
						drawShape(shape);

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
