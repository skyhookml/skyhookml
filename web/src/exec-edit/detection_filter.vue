<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Categories</label>
			<div class="col-sm-10">
				<table class="table">
					<tbody>
						<tr v-for="(category, i) in categories">
							<td>{{ category }}</td>
							<td>
								<button type="button" class="btn btn-danger" v-on:click="removeCategory(i)">Remove</button>
							</td>
						</tr>
						<tr>
							<td>
								<input v-model="addCategoryInput" type="text" class="form-control">
							</td>
							<td>
								<button type="button" class="btn btn-primary" v-on:click="addCategory">Add</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Score Threshold</label>
			<div class="col-sm-10">
				<input v-model.number="score" type="text" class="form-control">
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
			categories: [],
			score: 0,

			addCategoryInput: '',
		};
	},
	props: ['node'],
	created: function() {
		try {
			let s = JSON.parse(this.node.Params);
			this.categories = s.Categories;
			this.score = s.Score;
		} catch(e) {}
	},
	methods: {
		save: function() {
			let params = JSON.stringify({
				Categories: this.categories,
				Score: this.score,
			});
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: params,
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
		addCategory: function() {
			if(this.addCategoryInput === '') {
				return;
			}
			this.categories.push(this.addCategoryInput);
			this.addCategoryInput = '';
		},
		removeCategory: function(i) {
			this.categories.splice(i, 1);
		},
	},
};
</script>