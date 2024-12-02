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