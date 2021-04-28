<template>
<div>
	<template v-if="viewingItems != null">
		<render-item v-bind:dataType="dataset1.DataType" v-bind:item="viewingItems[0]"></render-item>
		<render-item v-bind:dataType="dataset2.DataType" v-bind:item="viewingItems[1]"></render-item>
		<div class="form-row align-items-center">
			<div class="col-auto">
				<button v-on:click="viewNext(-1)" type="button" class="btn btn-primary">Prev</button>
			</div>
			<div class="col-auto">
				<span>{{ viewingItems[0].Key }}</span>
				<span>({{ curIndex }} of {{ items.length }})</span>
			</div>
			<div class="col-auto">
				<button v-on:click="viewNext(1)" type="button" class="btn btn-primary">Next</button>
			</div>
		</div>
	</template>
</div>
</template>

<script>
import utils from './utils.js';
import RenderItem from './render-item.vue';

export default {
	components: {
		'render-item': RenderItem,
	},
	data: function() {
		return {
			dataset1: null,
			dataset2: null,
			items: [],
			curIndex: 0,
			viewingItems: null,
		};
	},
	created: function() {
		const nodeid1 = this.$route.params.nodeid;
		const nodeid2 = this.$route.params.othernodeid;
		Promise.all([
			utils.request(this, 'GET', '/exec-nodes/'+nodeid1+'/datasets', null, (data) => {
				this.dataset1 = data[0];
			}),
			utils.request(this, 'GET', '/exec-nodes/'+nodeid2+'/datasets', null, (data) => {
				this.dataset2 = data[0];
			}),
		]).then(() => {
			let items1 = {};
			let items2 = {};
			Promise.all([
				utils.request(this, 'GET', '/datasets/'+this.dataset1.ID+'/items', null, (items) => {
					items.forEach((item) => {
						items1[item.Key] = item;
					});
				}),
				utils.request(this, 'GET', '/datasets/'+this.dataset2.ID+'/items', null, (items) => {
					items.forEach((item) => {
						items2[item.Key] = item;
					});
				}),
			]).then(() => {
				for(let key in items1) {
					if(!items2[key]) {
						continue;
					}
					this.items.push([items1[key], items2[key]]);
				}
				if(this.items.length > 0) {
					this.curIndex = 0;
					this.viewingItems = this.items[this.curIndex];
				}
			});
		});
	},
	methods: {
		viewNext: function(direction) {
			this.curIndex = (this.curIndex+this.items.length+direction) % this.items.length;
			this.viewingItems = this.items[this.curIndex];
		},
	},
};
</script>
