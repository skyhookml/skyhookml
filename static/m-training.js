import utils from './utils.js';

import TrainNodeParents from './m-train-node-parents.js';

export default {
	components: {
		'm-train-node-parents': TrainNodeParents,
	},
	data: function() {
		return {
			meta: {},
			nodes: {},
			selectedNode: null,
			showingNewNodeModal: false,
			nodeRects: {},
			prevStage: null,
			resizeObserver: null,
		};
	},
	props: ['mtab'],
	// don't want to render until after mounted
	mounted: function() {
		this.update();
	},
	methods: {
		update: function() {
			Promise.all([
				utils.request(this, 'GET', '/kv/train-nodes-meta', null, (meta) => {
					if(meta) {
						this.meta = JSON.parse(meta);
					} else {
						this.meta = {};
					}
				}, null, {dataType: 'text'}),
				utils.request(this, 'GET', '/train-nodes', null, (nodes) => {
					this.nodes = {};
					nodes.forEach((node) => {
						this.nodes[node.ID] = node;
					});

					if(this.selectedNode) {
						if(this.nodes[this.selectedNode.ID]) {
							this.selectNode(this.nodes[this.selectedNode.ID]);
						} else {
							this.selectedNode = null;
						}
					}
				}),
			]).then(() => {
				this.render();
			});
		},
		render: function() {
			let dims = [1000, 500];
			let scale = (this.$refs.view.offsetWidth-10) / dims[0];

			if(this.prevStage) {
				this.prevStage.destroy();
			}
			if(this.resizeObserver) {
				this.resizeObserver.disconnect();
			}

			let stage = new Konva.Stage({
				container: this.$refs.layer,
				width: parseInt(dims[0]*scale),
				height: parseInt(dims[1]*scale),
			});
			this.prevStage = stage;

			let layer = new Konva.Layer();
			let rescaleLayer = () => {
				if(!this.$refs.view) {
					// probably editing node
					return;
				}
				let scale = (this.$refs.view.offsetWidth-10) / dims[0];
				stage.width(parseInt(dims[0]*scale));
				stage.height(parseInt(dims[1]*scale));
				layer.scaleX(scale);
				layer.scaleY(scale);
				layer.draw();
			};
			rescaleLayer();
			this.resizeObserver = new ResizeObserver(rescaleLayer);
			this.resizeObserver.observe(this.$refs.view);
			stage.add(layer);
			layer.add(new Konva.Rect({
				x: 0,
				y: 0,
				width: dims[0],
				height: dims[1],
				fill: 'lightgrey',
			}));

			let groups = {};
			let arrows = {};

			let save = () => {
				let meta = {};
				for(let gid in groups) {
					meta[gid] = [parseInt(groups[gid].x()), parseInt(groups[gid].y())];
				}
				utils.request(this, 'POST', '/kv/train-nodes-meta', {'val': JSON.stringify(meta)});
				this.meta = meta;
			};

			let addGroup = (id, text, meta) => {
				text = new Konva.Text({
					x: 0,
					y: 0,
					text: text,
					padding: 5,
				});
				text.offsetX(text.width() / 2);
				text.offsetY(text.height() / 2);
				let rect = new Konva.Rect({
					x: 0,
					y: 0,
					offsetX: text.offsetX(),
					offsetY: text.offsetY(),
					width: text.width(),
					height: text.height(),
					stroke: 'black',
					strokeWidth: 1,
					name: 'myrect',
				});
				let group = new Konva.Group({
					draggable: true,
					x: meta[0],
					y: meta[1],
				});
				group.mywidth = text.width();
				group.myheight = text.height();
				group.myrect = rect;
				group.on('dragend', save);
				group.add(rect);
				group.add(text);
				layer.add(group);
				groups[id] = group;
				return group;
			};

			let resetColors = () => {
				for(let gid in groups) {
					let rect = groups[gid].myrect;
					rect.fill('lightblue');
				}
				if(this.selectedNode != null) {
					groups['n'+this.selectedNode.ID].myrect.fill('salmon');
				}
				layer.draw();
			};

			// render the nodes
			let numDefault = 0;
			for(let nodeID in this.nodes) {
				let node = this.nodes[nodeID];
				let meta = this.meta['n' + nodeID];
				if(!meta) {
					meta = [500, 150+25*numDefault];
					numDefault++;
				}
				let group = addGroup('n'+nodeID, `${node.Name} (${node.Op})`, meta);
				let rect = group.myrect;

				group.on('mouseenter', () => {
					stage.container().style.cursor = 'pointer';
				})
				group.on('mouseleave', () => {
					stage.container().style.cursor = 'default';
				})
				group.on('click', (e) => {
					e.cancelBubble = true;
					this.selectNode(node);
					resetColors();
				});
			}

			resetColors();

			stage.on('click', (e) => {
				this.selectNode(null);
				resetColors();
			});

			// render the arrows
			let getClosestPoint = (group1, group2, isdst) => {
				let cx = group1.x();
				let cy = group1.y();
				let width = group1.mywidth;
				let height = group1.myheight;
				let padding = 0;
				if(isdst) {
					// add padding so arrow doesn't go into the rectangle
					padding = 3;
				}
				let p1 = [
					[cx, cy-height/2-padding],
					[cx, cy+height/2+padding],
					[cx-width/2-padding, cy],
					[cx+width/2+padding, cy],
				];
				let p2 = [group2.x(), group2.y()];
				let best = null;
				let bestDistance = 0;
				p1.forEach((p) => {
					let dx = p[0]-p2[0];
					let dy = p[1]-p2[1];
					let d = dx*dx+dy*dy;
					if(best == null || d < bestDistance) {
						best = p;
						bestDistance = d;
					}
				});
				return best;
			};
			for(let nodeID in this.nodes) {
				let node = this.nodes[nodeID];
				if(!node.ParentIDs) {
					continue;
				}
				let dst = 'n'+nodeID;
				node.ParentIDs.forEach((parentID) => {
					let src = 'n'+parentID;
					let p1 = getClosestPoint(groups[src], groups[dst], false);
					let p2 = getClosestPoint(groups[dst], groups[src], true);
					let arrow = new Konva.Arrow({
						points: [p1[0], p1[1], p2[0], p2[1]],
						pointerLength: 10,
						pointerWidth: 10,
						fill: 'black',
						stroke: 'black',
						strokeWidth: 2,
					});
					layer.add(arrow);
					if(!(src in arrows)) {
						arrows[src] = [];
					}
					if(!(dst in arrows)) {
						arrows[dst] = [];
					}
					arrows[src].push(['src', arrow, dst]);
					arrows[dst].push(['dst', arrow, src]);
				});
			}

			// add listeners to move the arrows when groups are dragged
			for(let gid in arrows) {
				let l = arrows[gid];
				groups[gid].on('dragmove', function() {
					l.forEach(function(el) {
						let mode = el[0];
						let arrow = el[1];
						let other = el[2];
						let p1, p2;
						if(mode == 'src') {
							p1 = getClosestPoint(groups[gid], groups[other], false);
							p2 = getClosestPoint(groups[other], groups[gid], true);
						} else {
							p1 = getClosestPoint(groups[other], groups[gid], false);
							p2 = getClosestPoint(groups[gid], groups[other], true);
						}
						let points = [p1[0], p1[1], p2[0], p2[1]];
						arrow.points(points);
						layer.draw();
					});
				});
			}

			layer.draw();
		},
		showNewNodeModal: function() {
			// if we're already showing it, we have to force re-create the component
			if(this.showingNewNodeModal) {
				this.showingNewNodeModal = false;
				Vue.nextTick(() => {
					this.showingNewNodeModal = true;
				});
			} else {
				this.showingNewNodeModal = true;
			}
		},
		onNewNodeModalClosed: function() {
			this.showingNewNodeModal = false;
			this.update();
		},
		selectNode: function(node) {
			this.selectedNode = node;
		},
		editNode: function() {
			this.$router.push('/ws/'+this.$route.params.ws+'/train/'+this.selectedNode.Op+'/'+this.selectedNode.ID);
		},
		runNode: function() {
			utils.request(this, 'POST', '/train-nodes/'+this.selectedNode.ID+'/run');
		},
		/*removeNode: function() {
			utils.request(this, 'POST', '/queries/node/remove', {id: this.selectedNode.ID}, () => {
				this.update();
			});
		},*/
		addParent: function(parentID) {
			let parentIDs;
			if(this.selectedNode.ParentIDs) {
				parentIDs = this.selectedNode.ParentIDs.concat([parentID]);
			} else {
				parentIDs = [parentID];
			}
			let params = JSON.stringify({ParentIDs: parentIDs});
			utils.request(this, 'POST', '/train-nodes/' + this.selectedNode.ID, params, () => {
				this.update();
			});
		},
		removeParent: function(idx) {
			let parentIDs = this.selectedNode.ParentIDs.filter((parentID, i) => i != idx);
			let params = JSON.stringify({ParentIDs: parentIDs});
			utils.request(this, 'POST', '/train-nodes/' + this.selectedNode.ID, params, () => {
				this.update();
			});
		},
	},
	watch: {
		mtab: function() {
			if(this.mtab != '#m-training-panel') {
				return;
			}
			this.update();
		},
	},
	template: `
<div style="height:100%" class="graph-div">
	<div ref="view" class="graph-view">
		<div ref="layer"></div>
	</div>
	<div>
		<div class="my-2">
			<button type="button" class="btn btn-primary" v-on:click="showNewNodeModal">New Node</button>
			<button type="button" class="btn btn-primary" :disabled="selectedNode == null" v-on:click="editNode">Edit Node</button>
			<button type="button" class="btn btn-primary" :disabled="selectedNode == null" v-on:click="runNode">Run Node</button>
		</div>
		<hr />
		<div v-if="selectedNode != null" class="my-2">
			<div>Node {{ selectedNode.Name }}</div>
			<!--<div><button type="button" class="btn btn-danger" v-on:click="removeNode">Remove Node</button></div>-->
			<div>
				<m-train-node-parents
					:node="selectedNode"
					:nodes="nodes"
					label="Parents"
					v-on:add="addParent($event)"
					v-on:remove="removeParent($event)"
					>
				</m-train-node-parents>
			</div>
		</div>
	</div>
	<m-add-train-node v-if="showingNewNodeModal" v-on:closed="onNewNodeModalClosed"></m-add-train-node>
</div>
	`,
};
