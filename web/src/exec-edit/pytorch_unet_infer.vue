<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<select-input-size v-model="params.Resize"></select-input-size>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
</template>

<script>
import utils from '../utils.js';
import SelectInputSize from './select-input-size.vue';

export default {
	components: {
		'select-input-size': SelectInputSize,
	},
	data: function() {
		return {
			params: null,
		};
	},
	props: ['node'],
	created: function() {
		let params = {};
		try {
			let s = JSON.parse(this.node.Params);
			params = s;
		} catch(e) {}
		if(!('Resize' in params)) {
			params.Resize = {
				Mode: 'keep',
				MaxDimension: 640,
				Width: 256,
				Height: 256,
				Multiple: 8,
			};
		}
		this.params = params;
	},
	methods: {
		save: function() {
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(this.params),
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
	},
};
</script>
