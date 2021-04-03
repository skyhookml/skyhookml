import utils from './utils.js';

export default {
	data: function() {
		return {
			archs: {},
			arch: null,
		};
	},
	props: ['node', 'value'],
	created: function() {
		this.arch = this.value;
		utils.request(this, 'GET', '/pytorch/archs', null, (archs) => {
			archs.forEach((arch) => {
				this.$set(this.archs, arch.ID, arch);
			});
		});
	},
	methods: {
		update: function() {
			this.$emit('input', this.arch);
		},
	},
	template: `
<div class="small-container">
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Architecture</label>
		<div class="col-sm-10">
			<select v-model.number="arch" class="form-select" @change="update">
				<template v-for="arch in archs">
					<option :key="arch.ID" :value="arch.ID">{{ arch.ID }}</option>
				</template>
			</select>
		</div>
	</div>
</div>
	`,
};
