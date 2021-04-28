<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Fraction</label>
			<div class="col-sm-10">
				<input v-model="fraction" type="text" class="form-control">
				<small class="form-text text-muted">
					Re-sample input data by this fraction. For example, 1/2 will sample every other element (or reduce video framerate by half).
				</small>
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			fraction: '',
		};
	},
	props: ['node'],
	created: function() {
		try {
			let s = JSON.parse(this.node.Params);
			this.fraction = s.Fraction;
		} catch(e) {}
	},
	methods: {
		save: function() {
			let params = JSON.stringify({
				Fraction: this.fraction,
			});
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: params,
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
	},
};
</script>