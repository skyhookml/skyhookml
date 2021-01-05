import utils from './utils.js';
import ExecNodeParents from './exec-node-parents.js';
import AddExecNode from './add-exec-node.js';

const Queries = {
	components: {
		'exec-node-parents': ExecNodeParents,
		'add-exec-node': AddExecNode,
	},
	data: function() {
		return {
			selectedDatasetID: null,
			datasets: {},
			meta: {},
			nodes: {},
			selectedNode: null,
			showingNewNodeModal: false,
			nodeRects: {},
			prevStage: null,
			resizeObserver: null,

			// for comparing
			workspaces: [],
			wsNodes: [],
			compareForm: {
				workspace: null,
				nodeID: null,
			}
		};
	},
	// don't want to render until after mounted
	mounted: function() {
		this.update();
	},
	methods: {
		update: function() {
			Promise.all([
				utils.request(this, 'GET', '/datasets', null, (data) => {
					let datasets = {};
					data.forEach((ds) => {
						datasets[ds.ID] = ds;
					});
					this.datasets = datasets;
				}),
				utils.request(this, 'GET', '/kv/exec-nodes-meta-'+this.$route.params.ws, null, (meta) => {
					if(meta) {
						this.meta = JSON.parse(meta);
					} else {
						this.meta = {};
					}
				}, null, {dataType: 'text'}),
				utils.request(this, 'GET', '/exec-nodes?ws='+this.$route.params.ws, null, (data) => {
					let nodes = {};
					data.forEach((node) => {
						nodes[node.ID] = node;
					});
					this.nodes = nodes;

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

			utils.request(this, 'GET', '/workspaces', null, (data) => {
				this.workspaces = data;
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
				utils.request(this, 'POST', '/kv/exec-nodes-meta-'+this.$route.params.ws, {'val': JSON.stringify(meta)});
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
					if(gid[0] == 'd') {
						rect.fill('lightgreen');
					} else {
						rect.fill('lightblue');
					}
				}
				if(this.selectedNode != null) {
					groups['n'+this.selectedNode.ID].myrect.fill('salmon');
				}
				layer.draw();
			};

			// (1) render the datasets
			let neededDatasetIDs = {};
			for(let nodeID in this.nodes) {
				this.nodes[nodeID].Parents.forEach((parent) => {
					if(parent.Type != 'd') {
						return;
					}
					neededDatasetIDs[parent.ID] = true;
				});
			}
			let datasets = [];
			for(let dsID in neededDatasetIDs) {
				datasets.push(this.datasets[dsID]);
			}
			let numDefault = 0;
			datasets.forEach((dataset) => {
				let meta = this.meta['d' + dataset.ID];
				if(!meta) {
					meta = [50+numDefault*200, 50];
					numDefault++;
				}
				addGroup('d'+dataset.ID, `Dataset ${dataset.Name}`, meta);
			});

			// (2) render the nodes
			numDefault = 0;
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
					this.selectNode(node, function() {
						resetColors();
					});
				});
			}

			resetColors();

			stage.on('click', (e) => {
				this.selectNode(null, function() {
					resetColors();
				});
			});

			// (3) render the arrows
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
				if(!node.Parents) {
					continue;
				}
				let dst = 'n'+nodeID;
				node.Parents.forEach((parent) => {
					let src;
					if(parent.Type == 'n') {
						src = 'n'+parent.ID;
					} else if(parent.Type == 'd') {
						src = 'd'+parent.ID;
					}
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

			// (4) add listeners to move the arrows when groups are dragged
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
		selectNode: function(node, cb) {
			if(!cb) {
				cb = function() {};
			}
			if(this.selectedNode) {
				this.selectedNode = null;
				Vue.nextTick(() => {
					this.selectedNode = node;
					cb();
				});
			} else {
				this.selectedNode = node;
				cb();
			}
		},
		editNode: function() {
			this.$router.push('/ws/'+this.$route.params.ws+'/exec/'+this.selectedNode.Op+'/'+this.selectedNode.ID);
		},
		runNode: function() {
			utils.request(this, 'POST', '/exec-nodes/'+this.selectedNode.ID+'/run');
		},
		deleteNode: function() {
			utils.request(this, 'DELETE', '/exec-nodes/'+this.selectedNode.ID, null, () => {
				this.update();
			});
		},
		addParent: function(parent, key) {
			let params = {};
			params[key] = this.selectedNode[key].concat([parent]);
			params = JSON.stringify(params);
			utils.request(this, 'POST', '/exec-nodes/' + this.selectedNode.ID, params, () => {
				this.update();
			});
		},
		removeParent: function(idx, key) {
			let params = {};
			params[key] = this.selectedNode[key].filter((parent, i) => i != idx);
			params = JSON.stringify(params);
			utils.request(this, 'POST', '/exec-nodes/' + this.selectedNode.ID, params, () => {
				this.update();
			});
		},
		selectCompareWorkspace: function() {
			this.compareForm.nodeID = null;
			this.wsNodes = null;
			utils.request(this, 'GET', '/exec-nodes?ws='+this.compareForm.workspace, null, (data) => {
				this.wsNodes = data;
			});
		},
		compareTo: function() {
			this.$router.push('/ws/'+this.$route.params.ws+'/compare/'+this.selectedNode.ID+'/'+this.compareForm.workspace+'/'+this.compareForm.nodeID);
		},
	},
	template: `
<div style="height:100%;" class="graph-div">
	<div class="graph-view" ref="view">
		<div ref="layer"></div>
	</div>
	<div>
		<div class="my-2">
			<button type="button" class="btn btn-primary" v-on:click="showNewNodeModal">New Node</button>
			<button type="button" class="btn btn-primary" :disabled="selectedNode == null" v-on:click="editNode">Edit Node</button>
			<button type="button" class="btn btn-primary" :disabled="selectedNode == null" v-on:click="runNode">Run Node</button>
			<button type="button" class="btn btn-danger" :disabled="selectedNode == null" v-on:click="deleteNode">Delete Node</button>
		</div>
		<hr />
		<div v-if="selectedNode != null" class="my-2">
			<div>Node {{ selectedNode.Name }}</div>
			<div>
				<exec-node-parents
					:node="selectedNode"
					pkey="Parents"
					:nodes="nodes"
					:datasets="datasets"
					label="Parents"
					v-on:add="addParent($event, 'Parents')"
					v-on:remove="removeParent($event, 'Parents')"
					>
				</exec-node-parents>
			</div>
			<div>
				<exec-node-parents
					:node="selectedNode"
					pkey="FilterParents"
					:nodes="nodes"
					:datasets="datasets"
					label="Filter Parents"
					v-on:add="addParent($event, 'FilterParents')"
					v-on:remove="removeParent($event, 'FilterParents')"
					>
				</exec-node-parents>
			</div>
			<div>
				<form v-on:submit.prevent="compareTo" class="form-inline my-2">
					<label class="ml-1">Compare to:</label>
					<select v-model="compareForm.workspace" @change="selectCompareWorkspace" class="form-control ml-1">
						<option v-for="ws in workspaces" :key="ws" :value="ws">{{ ws }}</option>
					</select>
					<select v-model="compareForm.nodeID" class="form-control ml-1">
						<option v-for="node in wsNodes" :key="node.ID" :value="node.ID">{{ node.Name }}</option>
					</select>
					<button type="submit" class="btn btn-primary ml-1">Go</button>
				</form>
			</div>
		</div>
	</div>
	<add-exec-node v-if="showingNewNodeModal" v-on:closed="onNewNodeModalClosed"></add-exec-node>
</div>
	`,
};
export default Queries;
