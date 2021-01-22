import utils from './utils.js';

const AnnotateInt = {
	data: function() {
		return {
			annoset: null,
			params: null,
			url: '',

			response: null,
			inputVal: '',

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
			if(!params.Range) {
				params.Range = 2;
			}
			this.params = params;
			utils.request(this, 'GET', this.url, null, this.update);
		});
	},
	mounted: function() {
		this.keypressHandler = (e) => {
			if(document.activeElement.tagName == 'INPUT') {
				return;
			}

			// keycode 48 through 57 are 0 through 9
			if(e.keyCode < 48 || e.keyCode > 57) {
				return;
			}
			var label = parseInt(e.keyCode) - 48;
			this.annotate(label);
		};
		this.$parent.$on('keypress', this.keypressHandler);
	},
	unmounted: function() {
		this.$parent.$off('keypress', this.keypressHandler);
		this.keypressHandler = null;
	},
	methods: {
		update: function(response) {
			this.response = response;
			this.inputVal = '';
			if(this.response.IsExisting) {
				utils.request(this, 'GET', '/datasets/'+this.annoset.Dataset.ID+'/items/'+this.response.Key+'/get', {format: 'json'}, (data) => {
					this.inputVal = data.toString();
				});
			}
		},
		getNew: function() {
			this.keyList = null;
			this.curIndex = 0;
			utils.request(this, 'GET', this.url, null, this.update);
		},
		getOld: function(i) {
			if(!this.keyList) {
				utils.request(this, 'GET', '/datasets/'+this.annoset.Dataset.ID+'/items', null, (items) => {
					this.keyList = items.map((item) => item.Key);
					this.getOld(0);
				});
				return;
			}

			this.curIndex = (i + this.keyList.length) % this.keyList.length;
			utils.request(this, 'GET', this.url+'?key='+this.keyList[this.curIndex], null, this.update);
		},
		annotate: function(val) {
			var request = {
				Key: this.response.Key,
				Data: JSON.stringify([val]),
				Format: 'json',
			};
			utils.request(this, 'POST', this.url, JSON.stringify(request), () => {
				if(this.keyList == null) {
					this.getNew();
				} else {
					this.getOld(this.curIndex+1);
				}
			});
		},
		annotateInput: function() {
			this.annotate(parseInt(this.inputVal));
		},
		saveParams: function() {
			utils.request(this, 'POST', '/annotate-datasets/'+this.annoset.ID, {Params: JSON.stringify(this.params)});
		},
	},
	template: `
<div>
	<template v-if="annoset != null">
		<div>
			<form class="form-inline" v-on:submit.prevent="saveParams">
				<label class="my-1 mx-1">Range</label>
				<input type="text" class="form-control my-1 mx-1" v-model="params.Range" />

				<button type="submit" class="btn btn-primary my-1 mx-1">Save Settings</button>
			</form>
		</div>
		<div>
			<template v-if="response != null">
				<img :src="'/datasets/'+annoset.Inputs[0].ID+'/items/'+response.Key+'/get?format=jpeg'" />
			</template>
		</div>
		<div class="form-row align-items-center">
			<div class="col-auto">
				<button v-on:click="getOld(curIndex-1)" type="button" class="btn btn-primary">Prev</button>
			</div>
			<div class="col-auto">
				<template v-if="response != null">
					<span>{{ response.Key }}</span>
					<span v-if="keyList != null">({{ curIndex }} of {{ keyList.length }})</span>
					<template v-if="inputVal">
						<span>(Value: {{ inputVal }})</span>
					</template>
				</template>
			</div>
			<div class="col-auto">
				<button v-on:click="getOld(curIndex+1)" type="button" class="btn btn-primary">Next</button>
			</div>
			<div class="col-auto">
				<button v-on:click="getNew" type="button" class="btn btn-primary">New</button>
			</div>
			<template v-if="parseInt(params.Range) > 0">
				<div v-for="i in parseInt(params.Range)">
					<button v-on:click="annotate(i-1)" type="button" class="btn btn-primary">{{ i-1 }}</button>
				</div>
			</template>
			<template v-else>
				<div class="col-auto">
					<form class="form-inline" v-on:submit.prevent="annotateInput">
						<input type="text" class="form-control" v-model="inputVal" />
						<button type="submit" class="btn btn-primary">Label</button>
					</form>
				</div>
			</template>
		</div>
	</template>
</div>
	`,
};
export default AnnotateInt;
