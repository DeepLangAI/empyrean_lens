function reformatJsonStr(jsonStr){
    try{
        let jsonObj = JSON.parse(jsonStr)
        // 如果是数组，则将数组中的每个元素都parse
        if (Array.isArray(jsonObj)){
            for (let i = 0; i < jsonObj.length; i++){
                try {
                    jsonObj[i] = JSON.parse(jsonObj[i])
                } catch (e) {
                    // do nothing
                }
            }
        }
        return JSON.stringify(jsonObj, null, 4)
    } catch (e) {
        return jsonStr.trim()
    }
    try {
        var jsonStr = JSON.stringify(JSON.parse(jsonStr), null, 4)
    } catch (e) {
        // 分别提取出json内容和非json内容，有可能在{}中，也有可能是[]中
        var jsonContent = jsonStr.match(/({.*})/g)
        if (jsonContent){
            try {
                var jsonStr = JSON.stringify(JSON.parse(jsonContent[0]), null, 4)
            } catch (e) {
                var jsonStr = jsonStr
            }
        }else{
            jsonContent = jsonStr.match(/(\[.*\])/g)
            if (jsonContent){
                try {
                    var jsonStr = JSON.stringify(JSON.parse(jsonContent[0]), null, 4)
                } catch (e) {
                    var jsonStr = jsonStr
                }
            }else{
                var jsonStr = value
            }
        }
    }
    jsonStr = jsonStr.trim()
    return jsonStr
}
