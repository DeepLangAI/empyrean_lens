class TreeNode {
    constructor(value) {
        this.value = value; // 节点的值
        this.children = []; // 子节点数组
    }

    // 添加子节点
    addChild(childNode) {
        this.children.push(childNode);
    }

    // 打印树的结构
    print(indent = 0) {
        console.log(' '.repeat(indent) + this.value);
        this.children.forEach(child => child.print(indent + 2));
    }
}

class Tree {
    constructor(rootValue) {
        this.root = new TreeNode(rootValue); // 初始化树的根节点
    }

    static parseSingleDoc_AbstractOutput(log){
        let tree = new Tree('root')
        let abstract = ''
        for (let i = 0; i < log.length; i ++){
            let subLog = log[i];
            if (subLog == "data too long"){
                continue
            }
            // if type is string
            if (typeof subLog === 'string') {
                subLog = JSON.parse(subLog)
            }

            if (subLog['text'] === undefined){
                continue
            }
            abstract += subLog['text'][0]
        }
        let child = new TreeNode(abstract)
        tree.root.addChild(child)
        return tree
    }
    static parseSingleDoc_ViewpointOutput(log){
        let tree = new Tree('root')
        for (let i = 0; i < log.length; i ++){
            let subLog = log[i];
            if (subLog == "data too long"){
                continue
            }
            // if type is string
            if (typeof subLog === 'string') {
                subLog = JSON.parse(subLog)
            }

            if (subLog['text'] === undefined){
                continue
            }
            let child = new TreeNode(subLog['text'][0])
            tree.root.addChild(child)
        }
        return tree
    }

    static parseSingleDoc_OutlineOutput(log){
        let tree = new Tree('root');
        let stack = []
        stack.push(tree.root)
        for (let i = 0; i < log.length; i++) {
            let subLog = log[i];
            if (subLog == "data too long"){
                continue
            }
            // if type is string
            if (typeof subLog === 'string') {
                subLog = JSON.parse(subLog)
            }
            if (subLog['id'] === undefined){
                continue
            }
            let level = subLog['id'].split('_').length
            console.log('parsing', i, stack.length, level)
            while (stack.length > level - 1) {
                stack.pop()
            }
            let parent = stack[stack.length - 1]
            let value = `【${subLog['infos']['headline']['text']}】${subLog['infos']['outline']['text']}`
            let child = new TreeNode(value)
            parent.addChild(child)
            stack.push(child)
            console.log('parsing', value)
        }
        return tree
    }
    static parseMultiDoc_OutlineOutput(log){
        let tree = new Tree('root');
        let stack = []
        stack.push(tree.root)
        for (let i = 0; i < log.length; i++) {
            let subLog = log[i];
            if (subLog == "data too long"){
                continue
            }
            // if type is string
            if (typeof subLog === 'string') {
                subLog = JSON.parse(subLog)
            }
            if (subLog['node_index'] === undefined || subLog['node_index'] === '0'){
                continue
            }
            let level = subLog['node_index'].split('_').length
            console.log('parsing', i, stack.length, level)
            while (stack.length > level - 1) {
                stack.pop()
            }
            let parent = stack[stack.length - 1]
            let value = `【${subLog['headline']}】${subLog['text']}`
            let child = new TreeNode(value)
            parent.addChild(child)
            stack.push(child)
            console.log('parsing', value)
        }
        return tree
    }
    static parseEduOutput(log){
        function tireTree(node){
            while (node.children.length == 1){
                node.children = node.children[0].children
            }
            for (let i = 0; i < node.children.length; i++){
                tireTree(node.children[i])
            }
            if (node.value == '虚拟节点'){
                if (node.children.length > 0){
                    node.value = node.children[0].value.slice(0, 20)
                }
            }
            let children = []
            for (let i = 0; i < node.children.length; i++){
                if (node.children[i].value != '虚拟节点'){
                    children.push(node.children[i])
                }
            }
            node.children = children
        }
        let tree = new Tree('root');
        let nodes = {}
        let links = {}
        let inDegree = {};

        for (let i = 0; i < log.length; i++) {
            let subLog = log[i];
            if (subLog == "data too long") {
                continue
            }
            // if type is string
            if (typeof subLog === 'string') {
                subLog = JSON.parse(subLog)
            }
            if (!(subLog['edu_tree_nodes'])){
                continue
            }
            for (let j = 0; j < subLog['edu_tree_nodes'].length; j++) {
                let node = subLog['edu_tree_nodes'][j]
                let content = node['content']
                if (content == 'figure'){
                    content = node['meta']['url']
                }
                let treeNode = new TreeNode(content)
                let nodeId = node['node_id']
                let parentId = node['parent_id']
                if (!(nodeId in nodes)){
                    nodes[nodeId] = treeNode
                }
                if (!(parentId in links)){
                    links[parentId] = []
                }
                links[parentId].push(nodeId)
                if (!(nodeId in inDegree)){
                    inDegree[nodeId] = 0
                }
                inDegree[nodeId] += 1
                if (!(parentId in inDegree)){
                    inDegree[parentId] = 0
                }
            }
        }
        // console.log(inDegree)
        // 遍历links，将节点添加到树中
        for (let parentId in links) {
            if (inDegree[parentId] == 0){
                let parentNode = nodes[parentId]
                if (!parentNode){
                    nodes[parentId] = new TreeNode('虚拟节点')
                }
                tree.root.addChild(nodes[parentId])
            }
            let children = links[parentId]
            for (let i = 0; i < children.length; i++) {
                let childId = children[i]
                let childNode = nodes[childId]
                let parentNode = nodes[parentId]
                if (!parentNode){
                } else{
                    parentNode.addChild(childNode)
                }
            }
        }
        if (tree.root.children.length == 1){
            tree.root = tree.root.children[0]
        }
        tireTree(tree.root)
        return tree
    }
    toDOM(node = this.root) {
        function helper(node, level=0){
            let li = $('<li>').addClass('node');
            if (node.children.length == 0){
                li.text(node.value);
                li.on('click', function(e) {
                    e.stopPropagation()
                })
            } else{
                const span = $('<span>').addClass('toggle').text(node.value);
                li.append(span);

                const $children = $('<ul>').addClass('collapse'); // 默认隐藏子节点
                node.children.forEach(child => {
                    const childDOM = helper(child, level + 1);
                    $children.append(childDOM);
                });
                li.append($children);

                if (level >= 1){
                    span.addClass('open');
                    $children.toggle()
                }
                // 点击事件
                li.on('click', function(e) {
                    e.stopPropagation()
                    $children.toggle(); // 展开或折叠子节点
                    span.toggleClass('open'); // 切换展开/折叠图标
                });
            }
            return li;
        }
        let $tree = $('<ul class="tree">')
        let $treeContent = helper(node)
        $tree.append($treeContent)
        return $tree
    }

    // 打印整棵树
    print() {
        this.root.print();
    }
}

// let tree = new Tree('')
// let output = tree.parseSingleDoc_OutlineOutput(data)
// output.print()
// console.log(output.root.children)