import utils from './utils.js';

export default {
	data: function() {
		return {
			node: null,
			trainNodes: {},
			trainNodeID: '',
		};
	},
	created: function() {
		utils.request(this, 'GET', '/train-nodes', null, (trainNodes) => {
			trainNodes.forEach((node) => {
				this.$set(this.trainNodes, node.ID, node);
			});
		});
		const nodeID = this.$route.params.nodeid;
		utils.request(this, 'GET', '/exec-nodes/'+nodeID, null, (node) => {
			this.node = node;
			try {
				let s = JSON.parse(this.node.Params);
				this.trainNodeID = s.TrainNodeID;
			} catch(e) {}
		});
	},
	methods: {
		save: function() {
			let params = {
				TrainNodeID: parseInt(this.trainNodeID),
			};
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(params),
			}));
		},
	},
	template: `
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Train Node</label>
			<div class="col-sm-10">
				<select v-model="trainNodeID" class="form-control">
					<template v-for="node in trainNodes">
						<option :key="node.ID" :value="node.ID">{{ node.Name }}</option>
					</template>
				</select>
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
