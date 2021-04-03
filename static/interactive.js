import utils from './utils.js';
import RenderItem from './render-item.js';

export default {
	components: {
		'render-item': RenderItem,
	},
	data: function() {
		return {
			node: null,
			interval: null,

			datasets: null,
			items: {},
			limit: 0,
		};
	},
	created: function() {
		const nodeID = this.$route.params.nodeid;
		utils.request(this, 'GET', '/exec-nodes/'+nodeID, null, (node) => {
			this.node = node;
			this.fetch();
			this.interval = setInterval(this.fetch, 1000);
		});
	},
	destroyed: function() {
		if(this.interval) {
			clearInterval(this.interval);
		}
	},
	methods: {
		fetch: function() {
			// fetch datasets first, in case the output names or dataset IDs have changed
			// then, fetch items in each dataset
			utils.request(this, 'GET', '/exec-nodes/'+this.node.ID+'/datasets', null, (datasets) => {
				// add output names not already in this.items
				let outputNames = Object.keys(datasets);
				for(let name of outputNames) {
					if(this.items[name]) {
						continue;
					}
					this.$set(this.items, name, []);
				}
				// unset names in this.items that no longer exist
				for(let name of Object.keys(this.items)) {
					if(outputNames.includes(name)) {
						continue;
					}
					delete this.items[name];
				}

				for(let [name, ds] of Object.entries(datasets)) {
					if(!ds) {
						continue;
					}
					utils.request(this, 'GET', '/datasets/'+ds.ID+'/items', null, (items) => {
						if(!items) {
							return;
						}
						this.items[name] = items;
					});
				}
			});
		},
		loadMore: function() {
			let minItems = Infinity;
			for(var name in this.items) {
				minItems = Math.min(minItems, this.items[name].length);
			}
			if(this.limit < minItems) {
				this.limit += 4;
				return;
			}
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID+'/incremental', {
				count: 4,
			});
			this.limit += 4;
		},
	},
	computed: {
		itemLists: function() {
			// group items into rows of 4 for each output name
			// also limit the number of items in each output to this.limit
			let items = {};
			for(var name in this.items) {
				let cur = this.items[name].slice(0, this.limit);
				let row = null;
				items[name] = [];
				cur.forEach((item) => {
					if(!row) {
						row = [];
						items[name].push(row);
					}
					row.push(item);
					if(row.length >= 4) {
						row = null;
					}
				});
			}
			return items;
		},
	},
	template: `
<div>
	<template v-if="node">
		<h2>Node: {{ node.Name }}</h2>
		<div v-for="(itemList, name) in itemLists">
			<h3>{{ name }}</h3>
			<div v-for="row in itemList" class="explore-results-row">
				<div v-for="item in row" class="explore-results-col">
					<template v-if="item.Dataset.DataType == 'video'">
						<video controls :src="'/datasets/'+item.Dataset.ID+'/items/'+item.Key+'/get?format=mp4'" class="explore-result-img"></video>
					</template>
					<template v-else-if="item.Dataset.DataType == 'image'">
						<img :src="'/datasets/'+item.Dataset.ID+'/items/'+item.Key+'/get?format=jpeg'" class="explore-result-img" />
					</template>
				</div>
			</div>
		</div>
		<button type="button" class="btn btn-primary" v-on:click="loadMore">Load More</button>
	</template>
</div>
	`,
};
