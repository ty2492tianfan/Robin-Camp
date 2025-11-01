## 本次实现
- 本次实现一个movie api， 并于movie api进行整合
    * POST /movies
    * GET /movies
    * POST /movies/{title}/ratings
    * GET /movies/{title}/rating
## 数据库选型
- 在此次项目中 我选择了PostgresSql作为本项目的数据库， 因为过去我有过Postgressql的经验，对此数据库较为熟悉
所以选择postgressql. 
### Movie Table
```sql
CREATE TABLE IF NOT EXISTS movies (
    id  BIGSERIAL PRIMARY KEY,   
    title TEXT NOT NULL UNIQUE,  
    genre TEXT NOT NULL,         
    distributor TEXT ,            
    release_date  DATE NOT NULL, 
    budget BIGINT,               
    mpa_rating TEXT,               
    box_office  JSONB              
);
```

### Rating Table
```sql
CREATE TABLE IF NOT EXISTS ratings (
    id  BIGSERIAL PRIMARY KEY,           -- 自增, 评论id
    movie_id  BIGINT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    rater_id  TEXT NOT NULL,
    rating  NUMERIC(2,1) NOT NULL CHECK (rating >= 0.5 AND rating <= 5.0), 
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (rating IN (0.5,1.0,1.5,2.0,2.5,3.0,3.5,4.0,4.5,5.0)),
    UNIQUE(movie_id, rater_id)
);
```

## 后端服务选型
选择go语言和gin 进行开发 gin可以简化很多的任务， 我初次接触go 语言， 便选择 官方教程使用go 和gin开发Api
https://go.dev/doc/tutorial/web-service-gin?utm_source=chatgpt.com
选择go+gin开发api较为方便,简化了很多任务。 
我将流程分成了多个部分， Movie_handle.go 负责处理movie/post 和movie/get 部分， 
Rate_handle.go 负责实现rate/post 和rate/get的部分， 
db.go 和auth.go是数据库和鉴权， movie/post 中用户成功后外接box Office, 如果
post成功的movie的其他数据在box Office中找到 则更新。 rate/post 中用户需鉴权 评分， 
用户同时也可以更新评分，并且取值范围为0.5 到5.0， 每个相隔为0.5. 途中遇到错误时返回错误并且给出错误信息
。 流程符合openApi 所要求的。 

##优化后续 
* 优化数据结构，处理异常数据，尤其是表格中的数据索引太多，应当设置更多约束
确保数据库的数据正确。 
* 简化逻辑，删除不必要的部分。 



